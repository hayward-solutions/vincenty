import Foundation
import MapLibre

/// Renders location history tracks as polylines on the MapLibre map.
///
/// Groups history entries by user_id and renders each user's track
/// as a colored polyline with per-user stable colors.
@MainActor
final class HistoryTracksController {

    private var mapView: MLNMapView?
    private var renderedSources: Set<String> = []
    private var userColors: [String: String] = [:]
    private var colorIndex = 0

    private static let colors = [
        "#3b82f6", "#ef4444", "#10b981", "#f59e0b",
        "#8b5cf6", "#ec4899", "#06b6d4", "#84cc16",
    ]

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    /// Update tracks with visible history entries.
    func update(entries: [LocationHistoryEntry]) {
        guard let mapView, let style = mapView.style else { return }

        // Remove old tracks
        removeAll()

        // Group entries by userId
        var byUser: [String: [LocationHistoryEntry]] = [:]
        for entry in entries {
            byUser[entry.userId, default: []].append(entry)
        }

        // Render each user's track
        for (userId, userEntries) in byUser {
            // Sort by time
            let sorted = userEntries.sorted { $0.recordedAt < $1.recordedAt }
            guard sorted.count >= 2 else { continue }

            let coords = sorted.map { [$0.lng, $0.lat] }
            let color = colorForUser(userId)

            let sourceId = "history-track-\(userId)"
            let lineLayerId = "\(sourceId)-line"
            let pointLayerId = "\(sourceId)-points"

            let geojson: [String: Any] = [
                "type": "FeatureCollection",
                "features": [
                    [
                        "type": "Feature",
                        "geometry": [
                            "type": "LineString",
                            "coordinates": coords,
                        ] as [String: Any],
                        "properties": [:] as [String: Any],
                    ] as [String: Any]
                ],
            ]

            guard let data = try? JSONSerialization.data(withJSONObject: geojson),
                  let shape = try? MLNShape(data: data, encoding: String.Encoding.utf8.rawValue)
            else { continue }

            let source = MLNShapeSource(identifier: sourceId, shape: shape, options: nil)
            style.addSource(source)

            // Line layer
            let line = MLNLineStyleLayer(identifier: lineLayerId, source: source)
            line.lineColor = NSExpression(forConstantValue: UIColor(hexString: color))
            line.lineWidth = NSExpression(forConstantValue: 3)
            line.lineOpacity = NSExpression(forConstantValue: 0.7)
            style.addLayer(line)

            // Endpoint circles
            let endpointGeoJSON: [String: Any] = [
                "type": "FeatureCollection",
                "features": [
                    pointFeature(sorted.first!),
                    pointFeature(sorted.last!),
                ],
            ]

            if let pointData = try? JSONSerialization.data(withJSONObject: endpointGeoJSON),
               let pointShape = try? MLNShape(
                data: pointData, encoding: String.Encoding.utf8.rawValue)
            {
                let pointSource = MLNShapeSource(
                    identifier: "\(sourceId)-pts", shape: pointShape, options: nil)
                style.addSource(pointSource)

                let circles = MLNCircleStyleLayer(identifier: pointLayerId, source: pointSource)
                circles.circleRadius = NSExpression(forConstantValue: 5)
                circles.circleColor = NSExpression(forConstantValue: UIColor(hexString: color))
                circles.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
                circles.circleStrokeWidth = NSExpression(forConstantValue: 2)
                style.addLayer(circles)

                renderedSources.insert("\(sourceId)-pts")
            }

            renderedSources.insert(sourceId)
        }
    }

    func removeAll() {
        guard let mapView, let style = mapView.style else { return }

        for sourceId in renderedSources {
            // Remove layers
            for suffix in ["-line", "-points"] {
                if let layer = style.layer(withIdentifier: "\(sourceId)\(suffix)") {
                    style.removeLayer(layer)
                }
            }
            if let source = style.source(withIdentifier: sourceId) {
                style.removeSource(source)
            }
        }
        renderedSources.removeAll()
    }

    // MARK: - Private

    private func colorForUser(_ userId: String) -> String {
        if let existing = userColors[userId] { return existing }
        let color = Self.colors[colorIndex % Self.colors.count]
        colorIndex += 1
        userColors[userId] = color
        return color
    }

    private func pointFeature(_ entry: LocationHistoryEntry) -> [String: Any] {
        [
            "type": "Feature",
            "geometry": [
                "type": "Point",
                "coordinates": [entry.lng, entry.lat],
            ] as [String: Any],
            "properties": [
                "username": entry.username,
                "time": entry.recordedAt,
            ] as [String: Any],
        ]
    }
}

// MARK: - UIColor Helper

private extension UIColor {
    convenience init(hexString: String) {
        var hex = hexString.trimmingCharacters(in: .whitespacesAndNewlines)
        hex = hex.hasPrefix("#") ? String(hex.dropFirst()) : hex

        var rgb: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&rgb)

        self.init(
            red: CGFloat((rgb >> 16) & 0xFF) / 255.0,
            green: CGFloat((rgb >> 8) & 0xFF) / 255.0,
            blue: CGFloat(rgb & 0xFF) / 255.0,
            alpha: 1.0)
    }
}
