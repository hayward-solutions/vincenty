import CoreLocation
import Foundation
import MapLibre

/// Handles interactive drawing on the MapLibre map (line/circle/rectangle).
///
/// Mirrors the web client's `draw-tool.tsx`:
/// - Tap to add points
/// - Live preview via a GeoJSON source + style layers
/// - Circle: 64-step polygon approximation
/// - Rectangle: oriented rectangle from 3 points (A, B define edge; C defines depth)
/// - Line: multi-point, double-tap to finalize
///
/// On iOS, map taps are received via the MapView coordinator's tap gesture
/// and forwarded here. The controller manages a `MLNShapeSource` for live preview.
@MainActor
final class DrawToolController {

    // MARK: - State

    private(set) var isActive = false
    private var mapView: MLNMapView?
    private var points: [CLLocationCoordinate2D] = []
    private var mode: DrawMode = .line
    private var style: DrawStyle = DrawStyle()

    /// Callback when a shape is completed.
    var onShapeComplete: ((GeoJSONFeature, DrawMode) -> Void)?

    // MARK: - Source/Layer IDs

    private static let sourceId = "draw-tool-geojson"
    private static let fillLayerId = "draw-tool-fill"
    private static let outlineLayerId = "draw-tool-outline"
    private static let lineLayerId = "draw-tool-line"
    private static let pointLayerId = "draw-tool-points"

    // MARK: - Setup

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    func activate(mode: DrawMode, style: DrawStyle) {
        self.mode = mode
        self.style = style
        self.points = []
        self.isActive = true
        setupSourceAndLayers()
    }

    func deactivate() {
        isActive = false
        points = []
        removeLayers()
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

    // MARK: - Tap Handling

    /// Called when the user taps the map while the draw tool is active.
    func handleTap(at coordinate: CLLocationCoordinate2D) {
        guard isActive else { return }

        points.append(coordinate)

        switch mode {
        case .line:
            // Lines are finalized by double-tap (handleDoubleTap)
            rebuildPreview()

        case .circle:
            if points.count == 2 {
                // Center + edge → complete
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
                // A, B, C → oriented rectangle
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

    // MARK: - Preview Rendering

    private func setupSourceAndLayers() {
        guard let mapView, let mapStyle = mapView.style else { return }

        // Remove existing if any
        removeLayers()

        // Add empty source
        let emptyGeoJSON = """
            {"type":"FeatureCollection","features":[]}
            """.data(using: .utf8)!
        guard let shape = try? MLNShape(data: emptyGeoJSON, encoding: String.Encoding.utf8.rawValue)
        else { return }

        let source = MLNShapeSource(identifier: Self.sourceId, shape: shape, options: nil)
        mapStyle.addSource(source)

        // Fill layer
        let fill = MLNFillStyleLayer(identifier: Self.fillLayerId, source: source)
        fill.fillColor = NSExpression(forConstantValue: UIColor(hex: style.fill))
        fill.fillOpacity = NSExpression(forConstantValue: style.fill == "transparent" ? 0 : 0.25)
        mapStyle.addLayer(fill)

        // Outline layer
        let outline = MLNLineStyleLayer(identifier: Self.outlineLayerId, source: source)
        outline.lineColor = NSExpression(forConstantValue: UIColor(hex: style.stroke))
        outline.lineWidth = NSExpression(forConstantValue: style.strokeWidth)
        mapStyle.addLayer(outline)

        // Line layer
        let line = MLNLineStyleLayer(identifier: Self.lineLayerId, source: source)
        line.lineColor = NSExpression(forConstantValue: UIColor(hex: style.stroke))
        line.lineWidth = NSExpression(forConstantValue: style.strokeWidth)
        mapStyle.addLayer(line)

        // Point layer (vertices)
        let point = MLNCircleStyleLayer(identifier: Self.pointLayerId, source: source)
        point.circleRadius = NSExpression(forConstantValue: 5)
        point.circleColor = NSExpression(forConstantValue: UIColor(hex: style.stroke))
        point.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
        point.circleStrokeWidth = NSExpression(forConstantValue: 2)
        mapStyle.addLayer(point)
    }

    private func rebuildPreview() {
        guard let mapView, let mapStyle = mapView.style,
              let source = mapStyle.source(withIdentifier: Self.sourceId) as? MLNShapeSource
        else { return }

        var features: [[String: Any]] = []

        // Add vertex points
        for point in points {
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "Point",
                    "coordinates": [point.longitude, point.latitude],
                ] as [String: Any],
                "properties": ["kind": "point"],
            ])
        }

        // Add preview shape based on mode and point count
        if mode == .line && points.count >= 2 {
            let coords = points.map { [$0.longitude, $0.latitude] }
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "LineString",
                    "coordinates": coords,
                ] as [String: Any],
                "properties": ["kind": "line"],
            ])
        } else if mode == .circle && points.count == 1 {
            // Just the center point — shown as vertex
        } else if mode == .rectangle && points.count == 2 {
            // Show the first edge as a line
            let coords = points.map { [$0.longitude, $0.latitude] }
            features.append([
                "type": "Feature",
                "geometry": [
                    "type": "LineString",
                    "coordinates": coords,
                ] as [String: Any],
                "properties": ["kind": "pending"],
            ])
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

    private func removeLayers() {
        guard let mapView, let mapStyle = mapView.style else { return }

        for layerId in [Self.fillLayerId, Self.outlineLayerId, Self.lineLayerId, Self.pointLayerId] {
            if let layer = mapStyle.layer(withIdentifier: layerId) {
                mapStyle.removeLayer(layer)
            }
        }
        if let source = mapStyle.source(withIdentifier: Self.sourceId) {
            mapStyle.removeSource(source)
        }
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

    /// Generate a circle polygon approximation (64 steps).
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

    /// Haversine distance between two coordinates in meters.
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

    /// Oriented rectangle from three points: A→B defines one edge, C defines perpendicular depth.
    private func orientedRectCoords(
        a: CLLocationCoordinate2D, b: CLLocationCoordinate2D, c: CLLocationCoordinate2D
    ) -> [CLLocationCoordinate2D] {
        let cosLat = cos(a.latitude * .pi / 180.0)

        // AB direction vector (in degrees, corrected for latitude)
        let abLat = b.latitude - a.latitude
        let abLng = (b.longitude - a.longitude) * cosLat

        // Perpendicular unit vector
        let abLen = sqrt(abLat * abLat + abLng * abLng)
        guard abLen > 0 else { return [a, b, b, a, a] }

        let perpLat = -abLng / abLen
        let perpLng = abLat / abLen

        // Project C onto the perpendicular to get depth
        let acLat = c.latitude - a.latitude
        let acLng = (c.longitude - a.longitude) * cosLat
        let depth = acLat * perpLat + acLng * perpLng

        // Offset in degrees (undo latitude correction for longitude)
        let offsetLat = depth * perpLat
        let offsetLng = depth * perpLng / cosLat

        let d = CLLocationCoordinate2D(latitude: a.latitude + offsetLat, longitude: a.longitude + offsetLng)
        let e = CLLocationCoordinate2D(latitude: b.latitude + offsetLat, longitude: b.longitude + offsetLng)

        return [a, b, e, d, a]
    }
}

// MARK: - UIColor Hex (private extension for draw tool)

private extension UIColor {
    convenience init(hex: String) {
        var hexSanitized = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        hexSanitized = hexSanitized.hasPrefix("#") ? String(hexSanitized.dropFirst()) : hexSanitized

        if hexSanitized == "transparent" || hexSanitized.isEmpty {
            self.init(white: 0, alpha: 0)
            return
        }

        var rgb: UInt64 = 0
        Scanner(string: hexSanitized).scanHexInt64(&rgb)

        self.init(
            red: CGFloat((rgb >> 16) & 0xFF) / 255.0,
            green: CGFloat((rgb >> 8) & 0xFF) / 255.0,
            blue: CGFloat(rgb & 0xFF) / 255.0,
            alpha: 1.0)
    }
}
