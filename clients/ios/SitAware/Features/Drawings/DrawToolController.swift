import CoreLocation
import Foundation
import MapLibre

/// Handles interactive drawing on the MapLibre map (line/circle/rectangle).
///
/// Mirrors the web client's `draw-tool.tsx`:
/// - Tap to add points
/// - Live annotation-based preview (MLNPolyline, MLNPointAnnotation)
/// - Circle: 64-step polygon approximation; completes on 2nd tap
/// - Rectangle: oriented rectangle from 3 points; completes on 3rd tap
/// - Line: multi-point, double-tap to finalize
///
/// Annotations are styled via Coordinator delegate methods in MapView.swift.
/// Annotation titles:
///   "draw-line"  — preview polyline + completed line/rectangle/circle outlines
///   "draw-fill"  — completed polygon fill
///   "draw-point" — vertex dots (in-progress only)
///
/// Two separate annotation arrays are maintained:
///   `annotations`          — in-progress preview (cleared on each tap/rebuild)
///   `completedAnnotations` — all completed shapes (persisted until deactivate or shape removed)
@MainActor
final class DrawToolController {

    // MARK: - State

    private(set) var isActive = false
    private var mapView: MLNMapView?
    private var points: [CLLocationCoordinate2D] = []
    private var mode: DrawMode = .line
    private var style: DrawStyle = DrawStyle()

    /// In-progress preview annotations (vertex dots + partial path).
    private var annotations: [MLNAnnotation] = []

    /// Completed shape annotations (persist until deactivate or caller removes shapes).
    private var completedAnnotations: [MLNAnnotation] = []

    /// Callback when a shape is completed.
    var onShapeComplete: ((GeoJSONFeature, DrawMode) -> Void)?

    // MARK: - Setup

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    func activate(mode: DrawMode, style: DrawStyle) {
        self.mode = mode
        self.style = style
        self.points = []
        self.isActive = true
        clearAnnotations()
        // Note: completed annotations are intentionally not cleared here —
        // callers call updateCompletedShapes() after activation to restore them.
    }

    func deactivate() {
        isActive = false
        points = []
        clearAnnotations()
        clearCompletedAnnotations()
    }

    func updateStyle(_ newStyle: DrawStyle) {
        self.style = newStyle
        rebuildPreview()
    }

    func updateMode(_ newMode: DrawMode) {
        self.mode = newMode
        self.points = []
        rebuildPreview()
    }

    // MARK: - Completed Shape Sync

    /// Called by MapScreen whenever `drawingsViewModel.completedShapes` changes.
    /// Re-renders all completed shapes as map annotations.
    func updateCompletedShapes(_ shapes: [CompletedShape]) {
        guard let mapView else { return }
        // Remove stale completed annotations
        if !completedAnnotations.isEmpty {
            mapView.removeAnnotations(completedAnnotations)
            completedAnnotations.removeAll()
        }
        // Re-add all current shapes
        for shape in shapes {
            let anns = buildAnnotations(for: shape.feature)
            completedAnnotations.append(contentsOf: anns)
        }
        if !completedAnnotations.isEmpty {
            mapView.addAnnotations(completedAnnotations)
        }
    }

    // MARK: - Tap Handling

    /// Called when the user taps the map while the draw tool is active.
    func handleTap(at coordinate: CLLocationCoordinate2D) {
        guard isActive else { return }

        points.append(coordinate)

        switch mode {
        case .line:
            // Lines are finalized by double-tap
            rebuildPreview()

        case .circle:
            if points.count == 2 {
                let center = points[0]
                let edge = points[1]
                let radius = haversineDistance(from: center, to: edge)
                let ring = generateCircleCoords(center: center, radiusMeters: radius, steps: 64)

                let feature = GeoJSONFeature(
                    geometry: .polygon([ring.map { [$0.longitude, $0.latitude] }]),
                    properties: featureProperties(shapeType: "circle", extras: [
                        "center": AnyCodable([center.longitude, center.latitude]),
                        "radiusMeters": AnyCodable(radius),
                    ]))

                onShapeComplete?(feature, .circle)
                points = []
                rebuildPreview()
            } else {
                rebuildPreview()
            }

        case .rectangle:
            if points.count == 3 {
                let ring = orientedRectCoords(a: points[0], b: points[1], c: points[2])

                let feature = GeoJSONFeature(
                    geometry: .polygon([ring.map { [$0.longitude, $0.latitude] }]),
                    properties: featureProperties(shapeType: "rectangle"))

                onShapeComplete?(feature, .rectangle)
                points = []
                rebuildPreview()
            } else {
                rebuildPreview()
            }
        }
    }

    /// Called on double-tap — finalizes a line.
    func handleDoubleTap(at coordinate: CLLocationCoordinate2D) {
        guard isActive, mode == .line, points.count >= 2 else { return }

        let coords = points.map { [$0.longitude, $0.latitude] }
        let feature = GeoJSONFeature(
            geometry: .lineString(coords),
            properties: featureProperties(shapeType: "line"))

        onShapeComplete?(feature, .line)
        points = []
        rebuildPreview()
    }

    // MARK: - Preview Rendering (in-progress only)

    private func rebuildPreview() {
        guard let mapView else { return }
        clearAnnotations()

        // Vertex dots
        for pt in points {
            let ann = MLNPointAnnotation()
            ann.coordinate = pt
            ann.title = "draw-point"
            annotations.append(ann)
        }

        // Preview line
        if mode == .line && points.count >= 2 {
            var coords = points
            let polyline = MLNPolyline(coordinates: &coords, count: UInt(coords.count))
            polyline.title = "draw-line"
            annotations.append(polyline)
        } else if mode == .rectangle && points.count == 2 {
            var coords = points
            let polyline = MLNPolyline(coordinates: &coords, count: UInt(coords.count))
            polyline.title = "draw-line"
            annotations.append(polyline)
        }

        if !annotations.isEmpty {
            mapView.addAnnotations(annotations)
        }
    }

    // MARK: - Completed Shape → Annotations

    private func buildAnnotations(for feature: GeoJSONFeature) -> [MLNAnnotation] {
        var result: [MLNAnnotation] = []

        switch feature.geometry {
        case .lineString(let coords):
            let clCoords = coords.map {
                CLLocationCoordinate2D(latitude: $0[1], longitude: $0[0])
            }
            var c = clCoords
            let polyline = MLNPolyline(coordinates: &c, count: UInt(c.count))
            polyline.title = "draw-line"
            result.append(polyline)

        case .polygon(let rings):
            guard let outer = rings.first, !outer.isEmpty else { break }
            let clCoords = outer.map {
                CLLocationCoordinate2D(latitude: $0[1], longitude: $0[0])
            }

            // Outline
            var outlineCoords = clCoords
            let outline = MLNPolyline(coordinates: &outlineCoords, count: UInt(outlineCoords.count))
            outline.title = "draw-line"
            result.append(outline)

            // Fill
            var fillCoords = clCoords
            let polygon = MLNPolygon(coordinates: &fillCoords, count: UInt(fillCoords.count))
            polygon.title = "draw-fill"
            result.append(polygon)

        case .point:
            break
        }

        return result
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

    private func clearCompletedAnnotations() {
        guard let mapView, !completedAnnotations.isEmpty else {
            completedAnnotations.removeAll()
            return
        }
        mapView.removeAnnotations(completedAnnotations)
        completedAnnotations.removeAll()
    }

    // MARK: - Geometry Helpers

    private func featureProperties(
        shapeType: String, extras: [String: AnyCodable] = [:]
    ) -> [String: AnyCodable] {
        var props: [String: AnyCodable] = [
            "kind": AnyCodable(shapeType == "line" ? "line" : "shape"),
            "shapeType": AnyCodable(shapeType),
            "stroke": AnyCodable(style.stroke),
            "fill": AnyCodable(style.fill),
            "strokeWidth": AnyCodable(style.strokeWidth),
        ]
        for (k, v) in extras { props[k] = v }
        return props
    }

    private func generateCircleCoords(
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

    private func haversineDistance(
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

    private func orientedRectCoords(
        a: CLLocationCoordinate2D, b: CLLocationCoordinate2D, c: CLLocationCoordinate2D
    ) -> [CLLocationCoordinate2D] {
        let cosLat = cos(a.latitude * .pi / 180.0)
        let abLat = b.latitude - a.latitude
        let abLng = (b.longitude - a.longitude) * cosLat
        let abLen = sqrt(abLat * abLat + abLng * abLng)
        guard abLen > 0 else { return [a, b, b, a, a] }
        let perpLat = -abLng / abLen
        let perpLng = abLat / abLen
        let acLat = c.latitude - a.latitude
        let acLng = (c.longitude - a.longitude) * cosLat
        let depth = acLat * perpLat + acLng * perpLng
        let offsetLat = depth * perpLat
        let offsetLng = depth * perpLng / cosLat
        let d = CLLocationCoordinate2D(
            latitude: a.latitude + offsetLat, longitude: a.longitude + offsetLng)
        let e = CLLocationCoordinate2D(
            latitude: b.latitude + offsetLat, longitude: b.longitude + offsetLng)
        return [a, b, e, d, a]
    }
}
