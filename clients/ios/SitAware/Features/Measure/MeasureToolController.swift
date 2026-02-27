import CoreLocation
import Foundation
import MapLibre

/// Measurement mode: line-distance or circle-radius.
enum MeasureMode: String, CaseIterable, Sendable {
    case line
    case circle
}

/// Result of a measurement operation.
struct MeasureResult: Sendable, Equatable {
    /// Individual segment distances in meters (line mode only).
    var segments: [Double] = []
    /// Total distance in meters (line mode: path length; circle mode: radius).
    var total: Double = 0
    /// Radius in meters (circle mode only).
    var radius: Double?
    /// Area in square meters (circle mode only).
    var area: Double?

    static let empty = MeasureResult()
}

/// Handles interactive measurement on the MapLibre map (point-to-point distance or circle radius/area).
///
/// Mirrors the web client's `measure-tool.tsx`:
/// - Line mode: tap to add points, each segment measured, total distance shown
/// - Circle mode: tap center, tap edge → radius + area
/// - Live annotation-based preview (MLNPolyline, MLNPolygon, MLNPointAnnotation)
///
/// Annotations are styled via Coordinator delegate methods in MapView.swift.
/// Annotation titles are used as identifiers for dispatch:
///   "measure-line"    — committed polyline path
///   "measure-outline" — circle outline polyline
///   "measure-radius"  — radius line from center to edge
///   "measure-fill"    — circle fill polygon
///   "measure-point"   — vertex dots (MLNPointAnnotation)
///   "measure-center"  — center dot in circle mode (MLNPointAnnotation)
///
/// On iOS, map taps are forwarded from the MapView coordinator.
@MainActor
final class MeasureToolController {

    // MARK: - State

    private(set) var isActive = false
    private var mapView: MLNMapView?
    private var points: [CLLocationCoordinate2D] = []
    private var mode: MeasureMode = .line

    /// All annotations currently shown on the map for this tool.
    private var annotations: [MLNAnnotation] = []

    /// Callback when measurements change (segments + total + radius/area).
    var onMeasurementsChange: ((MeasureResult) -> Void)?

    // MARK: - Setup

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    func activate(mode: MeasureMode) {
        self.mode = mode
        self.points = []
        self.isActive = true
        clearAnnotations()
        onMeasurementsChange?(.empty)
    }

    func deactivate() {
        isActive = false
        points = []
        clearAnnotations()
        onMeasurementsChange?(.empty)
    }

    func updateMode(_ newMode: MeasureMode) {
        self.mode = newMode
        self.points = []
        rebuildPreview()
        onMeasurementsChange?(.empty)
    }

    func clear() {
        points = []
        rebuildPreview()
        onMeasurementsChange?(.empty)
    }

    // MARK: - Tap Handling

    /// Called when the user taps the map while the measure tool is active.
    func handleTap(at coordinate: CLLocationCoordinate2D) {
        guard isActive else { return }

        switch mode {
        case .line:
            points.append(coordinate)
            rebuildPreview()
            emitMeasurements()

        case .circle:
            points.append(coordinate)
            if points.count == 2 {
                // Center + edge → measure
                rebuildPreview()
                emitMeasurements()
            } else if points.count > 2 {
                // Reset on third tap (start new circle)
                points = [coordinate]
                rebuildPreview()
                onMeasurementsChange?(.empty)
            } else {
                rebuildPreview()
            }
        }
    }

    /// Called on double-tap — finalizes a line measurement (prevents further extension).
    func handleDoubleTap(at coordinate: CLLocationCoordinate2D) {
        guard isActive, mode == .line, points.count >= 2 else { return }
        // Double-tap fires single-tap first; if it added a duplicate point, remove it.
        if let last = points.last,
           last.latitude == coordinate.latitude && last.longitude == coordinate.longitude
        {
            points.removeLast()
        }
        rebuildPreview()
        emitMeasurements()
    }

    // MARK: - Measurement Calculations

    private func emitMeasurements() {
        switch mode {
        case .line:
            guard points.count >= 2 else {
                onMeasurementsChange?(.empty)
                return
            }
            var segments: [Double] = []
            for i in 1..<points.count {
                segments.append(Self.haversineDistance(from: points[i - 1], to: points[i]))
            }
            let total = segments.reduce(0, +)
            onMeasurementsChange?(MeasureResult(segments: segments, total: total))

        case .circle:
            guard points.count >= 2 else {
                onMeasurementsChange?(.empty)
                return
            }
            let radius = Self.haversineDistance(from: points[0], to: points[1])
            let area = Double.pi * radius * radius
            onMeasurementsChange?(MeasureResult(
                segments: [],
                total: radius,
                radius: radius,
                area: area))
        }
    }

    // MARK: - Preview Rendering (Annotation-Based)

    private func rebuildPreview() {
        guard let mapView else { return }
        clearAnnotations()

        switch mode {
        case .line:   buildLineAnnotations()
        case .circle: buildCircleAnnotations()
        }

        if !annotations.isEmpty {
            mapView.addAnnotations(annotations)
        }
    }

    private func buildLineAnnotations() {
        // Vertex dots
        for point in points {
            let ann = MLNPointAnnotation()
            ann.coordinate = point
            ann.title = "measure-point"
            annotations.append(ann)
        }

        // Committed polyline connecting all tapped points
        if points.count >= 2 {
            var coords = points
            let polyline = MLNPolyline(coordinates: &coords, count: UInt(coords.count))
            polyline.title = "measure-line"
            annotations.append(polyline)
        }
    }

    private func buildCircleAnnotations() {
        guard !points.isEmpty else { return }

        // Center dot
        let centerAnn = MLNPointAnnotation()
        centerAnn.coordinate = points[0]
        centerAnn.title = "measure-center"
        annotations.append(centerAnn)

        guard points.count >= 2 else { return }

        let center = points[0]
        let edge = points[1]
        let radius = Self.haversineDistance(from: center, to: edge)
        let ring = Self.generateCircleCoords(center: center, radiusMeters: radius, steps: 64)

        // Fill polygon
        var ringCoords = ring
        let polygon = MLNPolygon(coordinates: &ringCoords, count: UInt(ringCoords.count))
        polygon.title = "measure-fill"
        annotations.append(polygon)

        // Outline polyline (closed ring)
        var outlineCoords = ring
        let outline = MLNPolyline(coordinates: &outlineCoords, count: UInt(outlineCoords.count))
        outline.title = "measure-outline"
        annotations.append(outline)

        // Radius line from center to edge
        var radiusCoords = [center, edge]
        let radiusLine = MLNPolyline(coordinates: &radiusCoords, count: 2)
        radiusLine.title = "measure-radius"
        annotations.append(radiusLine)
    }

    // MARK: - Annotation Lifecycle

    private func clearAnnotations() {
        guard let mapView, !annotations.isEmpty else {
            annotations.removeAll()
            return
        }
        mapView.removeAnnotations(annotations)
        annotations.removeAll()
    }

    // MARK: - Geometry (static, reusable by other controllers)

    /// Haversine distance between two coordinates in meters.
    static func haversineDistance(
        from: CLLocationCoordinate2D, to: CLLocationCoordinate2D
    ) -> Double {
        let R = 6_371_000.0
        let dLat = (to.latitude - from.latitude) * .pi / 180.0
        let dLng = (to.longitude - from.longitude) * .pi / 180.0
        let a = sin(dLat / 2) * sin(dLat / 2)
            + cos(from.latitude * .pi / 180.0) * cos(to.latitude * .pi / 180.0)
            * sin(dLng / 2) * sin(dLng / 2)
        return R * 2 * atan2(sqrt(a), sqrt(1 - a))
    }

    /// Midpoint between two coordinates.
    static func midpoint(
        _ a: CLLocationCoordinate2D, _ b: CLLocationCoordinate2D
    ) -> CLLocationCoordinate2D {
        CLLocationCoordinate2D(
            latitude: (a.latitude + b.latitude) / 2,
            longitude: (a.longitude + b.longitude) / 2)
    }

    /// Generate a circle polygon approximation.
    static func generateCircleCoords(
        center: CLLocationCoordinate2D, radiusMeters: Double, steps: Int
    ) -> [CLLocationCoordinate2D] {
        let earthRadius = 6_371_000.0
        let dLat = (radiusMeters / earthRadius) * (180.0 / .pi)
        let dLng = dLat / cos(center.latitude * .pi / 180.0)

        var coords: [CLLocationCoordinate2D] = []
        for i in 0...steps {
            let angle = Double(i) * (2.0 * .pi / Double(steps))
            let lat = center.latitude + dLat * sin(angle)
            let lng = center.longitude + dLng * cos(angle)
            coords.append(CLLocationCoordinate2D(latitude: lat, longitude: lng))
        }
        return coords
    }

    /// Format a distance in meters for display.
    static func formatDistance(_ meters: Double) -> String {
        if meters >= 1000 {
            return String(format: "%.2fkm", meters / 1000)
        } else {
            return String(format: "%.0fm", meters)
        }
    }

    /// Format an area in square meters for display.
    static func formatArea(_ squareMeters: Double) -> String {
        if squareMeters >= 1_000_000 {
            return String(format: "%.2fkm\u{00B2}", squareMeters / 1_000_000)
        } else {
            return String(format: "%.0fm\u{00B2}", squareMeters)
        }
    }
}
