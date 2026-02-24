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
/// - Live GeoJSON preview with styled layers
/// - Distance labels at segment midpoints
///
/// On iOS, map taps are forwarded from the MapView coordinator.
@MainActor
final class MeasureToolController {

    // MARK: - State

    private(set) var isActive = false
    private var mapView: MLNMapView?
    private var points: [CLLocationCoordinate2D] = []
    private var mode: MeasureMode = .line

    /// Callback when measurements change (segments + total + radius/area).
    var onMeasurementsChange: ((MeasureResult) -> Void)?

    // MARK: - Source/Layer IDs

    private static let sourceId = "measure-geojson"
    // Line mode layers
    private static let linesLayerId = "measure-lines"
    private static let pendingLayerId = "measure-pending"
    private static let pointsLayerId = "measure-points"
    private static let labelsLayerId = "measure-labels"
    // Circle mode layers
    private static let fillLayerId = "measure-fill"
    private static let outlineLayerId = "measure-outline"
    private static let radiusLayerId = "measure-radius"
    private static let centerLayerId = "measure-center"
    private static let circleLabelLayerId = "measure-circle-labels"

    // MARK: - Setup

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    func activate(mode: MeasureMode) {
        self.mode = mode
        self.points = []
        self.isActive = true
        setupSourceAndLayers()
        onMeasurementsChange?(.empty)
    }

    func deactivate() {
        isActive = false
        points = []
        removeLayers()
        onMeasurementsChange?(.empty)
    }

    func updateMode(_ newMode: MeasureMode) {
        self.mode = newMode
        self.points = []
        removeLayers()
        setupSourceAndLayers()
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
        // Double-tap adds a duplicate — remove it
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

    // MARK: - Preview Rendering

    private func setupSourceAndLayers() {
        guard let mapView, let mapStyle = mapView.style else { return }

        removeLayers()

        // Add empty GeoJSON source
        let emptyGeoJSON = """
            {"type":"FeatureCollection","features":[]}
            """.data(using: .utf8)!
        guard let shape = try? MLNShape(data: emptyGeoJSON, encoding: String.Encoding.utf8.rawValue)
        else { return }

        let source = MLNShapeSource(identifier: Self.sourceId, shape: shape, options: nil)
        mapStyle.addSource(source)

        switch mode {
        case .line:
            setupLineLayers(source: source, style: mapStyle)
        case .circle:
            setupCircleLayers(source: source, style: mapStyle)
        }
    }

    private func setupLineLayers(source: MLNShapeSource, style: MLNStyle) {
        // Committed lines (solid)
        let lines = MLNLineStyleLayer(identifier: Self.linesLayerId, source: source)
        lines.lineColor = NSExpression(forConstantValue: UIColor.systemBlue)
        lines.lineWidth = NSExpression(forConstantValue: 3)
        lines.predicate = NSPredicate(format: "kind == 'line'")
        style.addLayer(lines)

        // Pending segment (dashed)
        let pending = MLNLineStyleLayer(identifier: Self.pendingLayerId, source: source)
        pending.lineColor = NSExpression(forConstantValue: UIColor.systemBlue.withAlphaComponent(0.6))
        pending.lineWidth = NSExpression(forConstantValue: 2)
        pending.lineDashPattern = NSExpression(forConstantValue: [4, 4])
        pending.predicate = NSPredicate(format: "kind == 'pending'")
        style.addLayer(pending)

        // Points
        let pointLayer = MLNCircleStyleLayer(identifier: Self.pointsLayerId, source: source)
        pointLayer.circleRadius = NSExpression(forConstantValue: 5)
        pointLayer.circleColor = NSExpression(forConstantValue: UIColor.systemBlue)
        pointLayer.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
        pointLayer.circleStrokeWidth = NSExpression(forConstantValue: 2)
        pointLayer.predicate = NSPredicate(format: "kind == 'point'")
        style.addLayer(pointLayer)

        // Distance labels
        let labels = MLNSymbolStyleLayer(identifier: Self.labelsLayerId, source: source)
        labels.text = NSExpression(forKeyPath: "label")
        labels.textFontSize = NSExpression(forConstantValue: 12)
        labels.textColor = NSExpression(forConstantValue: UIColor.white)
        labels.textHaloColor = NSExpression(forConstantValue: UIColor.black)
        labels.textHaloWidth = NSExpression(forConstantValue: 1.5)
        labels.textOffset = NSExpression(forConstantValue: NSValue(cgVector: CGVector(dx: 0, dy: -1.2)))
        labels.predicate = NSPredicate(format: "kind == 'label'")
        style.addLayer(labels)
    }

    private func setupCircleLayers(source: MLNShapeSource, style: MLNStyle) {
        // Fill
        let fill = MLNFillStyleLayer(identifier: Self.fillLayerId, source: source)
        fill.fillColor = NSExpression(forConstantValue: UIColor.systemBlue.withAlphaComponent(0.1))
        fill.fillOpacity = NSExpression(forConstantValue: 1)
        fill.predicate = NSPredicate(format: "kind == 'circle-fill'")
        style.addLayer(fill)

        // Outline
        let outline = MLNLineStyleLayer(identifier: Self.outlineLayerId, source: source)
        outline.lineColor = NSExpression(forConstantValue: UIColor.systemBlue)
        outline.lineWidth = NSExpression(forConstantValue: 2)
        outline.predicate = NSPredicate(format: "kind == 'circle-outline'")
        style.addLayer(outline)

        // Radius line (dashed)
        let radius = MLNLineStyleLayer(identifier: Self.radiusLayerId, source: source)
        radius.lineColor = NSExpression(forConstantValue: UIColor.systemBlue.withAlphaComponent(0.7))
        radius.lineWidth = NSExpression(forConstantValue: 2)
        radius.lineDashPattern = NSExpression(forConstantValue: [6, 4])
        radius.predicate = NSPredicate(format: "kind == 'radius'")
        style.addLayer(radius)

        // Center point
        let center = MLNCircleStyleLayer(identifier: Self.centerLayerId, source: source)
        center.circleRadius = NSExpression(forConstantValue: 6)
        center.circleColor = NSExpression(forConstantValue: UIColor.systemBlue)
        center.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
        center.circleStrokeWidth = NSExpression(forConstantValue: 2)
        center.predicate = NSPredicate(format: "kind == 'center'")
        style.addLayer(center)

        // Radius label
        let label = MLNSymbolStyleLayer(identifier: Self.circleLabelLayerId, source: source)
        label.text = NSExpression(forKeyPath: "label")
        label.textFontSize = NSExpression(forConstantValue: 12)
        label.textColor = NSExpression(forConstantValue: UIColor.white)
        label.textHaloColor = NSExpression(forConstantValue: UIColor.black)
        label.textHaloWidth = NSExpression(forConstantValue: 1.5)
        label.textOffset = NSExpression(forConstantValue: NSValue(cgVector: CGVector(dx: 0, dy: -1.2)))
        label.predicate = NSPredicate(format: "kind == 'label'")
        style.addLayer(label)
    }

    private func rebuildPreview() {
        guard let mapView, let mapStyle = mapView.style,
              let source = mapStyle.source(withIdentifier: Self.sourceId) as? MLNShapeSource
        else { return }

        var features: [[String: Any]] = []

        switch mode {
        case .line:
            features = buildLineFeatures()
        case .circle:
            features = buildCircleFeatures()
        }

        let collection: [String: Any] = [
            "type": "FeatureCollection",
            "features": features,
        ]

        if let data = try? JSONSerialization.data(withJSONObject: collection),
           let shape = try? MLNShape(data: data, encoding: String.Encoding.utf8.rawValue)
        {
            source.shape = shape
        }
    }

    // MARK: - GeoJSON Feature Builders

    private func buildLineFeatures() -> [[String: Any]] {
        var features: [[String: Any]] = []

        // Vertex points
        for point in points {
            features.append(pointFeature(coordinate: point, kind: "point"))
        }

        // Committed line segments
        if points.count >= 2 {
            let coords = points.map { [$0.longitude, $0.latitude] }
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "LineString",
                    "coordinates": coords,
                ] as [String: Any],
                "properties": ["kind": "line"] as [String: Any],
            ])
        }

        // Distance labels at midpoints of each segment
        if points.count >= 2 {
            for i in 1..<points.count {
                let dist = Self.haversineDistance(from: points[i - 1], to: points[i])
                let mid = Self.midpoint(points[i - 1], points[i])
                features.append([
                    "type": "Feature",
                    "geometry": [
                        "type": "Point",
                        "coordinates": [mid.longitude, mid.latitude],
                    ] as [String: Any],
                    "properties": [
                        "kind": "label",
                        "label": Self.formatDistance(dist),
                    ] as [String: Any],
                ])
            }

            // Total distance label at the last point (if > 1 segment)
            if points.count > 2, let last = points.last {
                let total = (1..<points.count).reduce(0.0) { sum, i in
                    sum + Self.haversineDistance(from: points[i - 1], to: points[i])
                }
                features.append([
                    "type": "Feature",
                    "geometry": [
                        "type": "Point",
                        "coordinates": [last.longitude, last.latitude],
                    ] as [String: Any],
                    "properties": [
                        "kind": "label",
                        "label": "Total: \(Self.formatDistance(total))",
                    ] as [String: Any],
                ])
            }
        }

        return features
    }

    private func buildCircleFeatures() -> [[String: Any]] {
        var features: [[String: Any]] = []

        guard !points.isEmpty else { return features }

        // Center point
        features.append(pointFeature(coordinate: points[0], kind: "center"))

        if points.count >= 2 {
            let center = points[0]
            let edge = points[1]
            let radius = Self.haversineDistance(from: center, to: edge)

            // Circle polygon (64-step approximation)
            let ring = Self.generateCircleCoords(center: center, radiusMeters: radius, steps: 64)
            let ringCoords = ring.map { [$0.longitude, $0.latitude] }

            // Fill
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "Polygon",
                    "coordinates": [ringCoords],
                ] as [String: Any],
                "properties": ["kind": "circle-fill"] as [String: Any],
            ])

            // Outline
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "Polygon",
                    "coordinates": [ringCoords],
                ] as [String: Any],
                "properties": ["kind": "circle-outline"] as [String: Any],
            ])

            // Radius line (dashed, center → edge)
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "LineString",
                    "coordinates": [
                        [center.longitude, center.latitude],
                        [edge.longitude, edge.latitude],
                    ],
                ] as [String: Any],
                "properties": ["kind": "radius"] as [String: Any],
            ])

            // Radius label at midpoint
            let mid = Self.midpoint(center, edge)
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "Point",
                    "coordinates": [mid.longitude, mid.latitude],
                ] as [String: Any],
                "properties": [
                    "kind": "label",
                    "label": Self.formatDistance(radius),
                ] as [String: Any],
            ])
        }

        return features
    }

    // MARK: - Helpers

    private func pointFeature(coordinate: CLLocationCoordinate2D, kind: String) -> [String: Any] {
        [
            "type": "Feature",
            "geometry": [
                "type": "Point",
                "coordinates": [coordinate.longitude, coordinate.latitude],
            ] as [String: Any],
            "properties": ["kind": kind] as [String: Any],
        ]
    }

    private func removeLayers() {
        guard let mapView, let mapStyle = mapView.style else { return }

        let allLayerIds = [
            Self.linesLayerId, Self.pendingLayerId, Self.pointsLayerId, Self.labelsLayerId,
            Self.fillLayerId, Self.outlineLayerId, Self.radiusLayerId, Self.centerLayerId,
            Self.circleLabelLayerId,
        ]

        for layerId in allLayerIds {
            if let layer = mapStyle.layer(withIdentifier: layerId) {
                mapStyle.removeLayer(layer)
            }
        }
        if let source = mapStyle.source(withIdentifier: Self.sourceId) {
            mapStyle.removeSource(source)
        }
    }

    // MARK: - Geometry (static, reusable)

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

    /// Generate a circle polygon approximation (64 steps).
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
