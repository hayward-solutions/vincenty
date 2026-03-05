import Foundation
import MapLibre

/// Renders location history tracks as polylines on the MapLibre map.
///
/// Uses a two-phase update strategy mirroring the web client's `history-tracks.tsx`:
///
/// 1. `setupLayers(allEntries:)` — called once when the full history dataset changes.
///    Groups entries by user, creates stable MapLibre sources + layers (seeded with
///    empty geometry). Expensive layer add/remove only happens here, not every frame.
///
/// 2. `updateData(visibleEntries:)` — called on every animation frame (30fps).
///    Updates each source's shape in-place via `source.shape =` so MapLibre layers
///    are never torn down mid-playback. This is the efficient path.
///
/// Tracks with only one location point still get a head-marker dot so users with
/// sparse history are not invisible.
@MainActor
final class HistoryTracksController {

    // MARK: - State

    private var mapView: MLNMapView?

    /// Track line sources keyed by userId.
    private var trackSources: [String: MLNShapeSource] = [:]
    /// Head marker sources keyed by userId (one circle per track head).
    private var headSources: [String: MLNShapeSource] = [:]

    /// Stable color assignment per userId.
    private var userColors: [String: UIColor] = [:]
    private var colorIndex = 0

    private static let trackColors: [UIColor] = [
        UIColor(hexString: "#3b82f6"),
        UIColor(hexString: "#ef4444"),
        UIColor(hexString: "#10b981"),
        UIColor(hexString: "#f59e0b"),
        UIColor(hexString: "#8b5cf6"),
        UIColor(hexString: "#ec4899"),
        UIColor(hexString: "#06b6d4"),
        UIColor(hexString: "#84cc16"),
    ]

    // MARK: - Attach

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    // MARK: - Phase 1: Layer lifecycle

    /// Called when `historyEntries` changes (once per replay session, not per frame).
    /// Removes all old sources/layers and creates stable empty ones for each user track.
    func setupLayers(allEntries: [LocationHistoryEntry]) {
        guard let mapView, let style = mapView.style else { return }

        removeAll()

        guard !allEntries.isEmpty else { return }

        // Group all entries by userId (no time filtering here).
        var userIds: [String] = []
        var seen = Set<String>()
        for entry in allEntries {
            if seen.insert(entry.userId).inserted {
                userIds.append(entry.userId)
            }
        }

        // Create one source + line layer + head source + head layer per user.
        for userId in userIds {
            let color = colorForUser(userId)

            // — Track line —
            let lineSourceId = trackLineSourceId(userId)
            let lineLayerId  = trackLineLayerId(userId)
            let emptyLine    = MLNShapeCollectionFeature(shapes: [])
            let lineSource   = MLNShapeSource(identifier: lineSourceId, shape: emptyLine, options: nil)
            style.addSource(lineSource)
            trackSources[userId] = lineSource

            let lineLayer         = MLNLineStyleLayer(identifier: lineLayerId, source: lineSource)
            lineLayer.lineColor   = NSExpression(forConstantValue: color)
            lineLayer.lineWidth   = NSExpression(forConstantValue: 3)
            lineLayer.lineOpacity = NSExpression(forConstantValue: 0.8)
            lineLayer.lineJoin    = NSExpression(forConstantValue: "round")
            lineLayer.lineCap     = NSExpression(forConstantValue: "round")
            style.addLayer(lineLayer)

            // — Head marker (circle at leading edge) —
            let headSourceId = trackHeadSourceId(userId)
            let headLayerId  = trackHeadLayerId(userId)
            let emptyPoint   = MLNShapeCollectionFeature(shapes: [])
            let headSource   = MLNShapeSource(identifier: headSourceId, shape: emptyPoint, options: nil)
            style.addSource(headSource)
            headSources[userId] = headSource

            let headLayer               = MLNCircleStyleLayer(identifier: headLayerId, source: headSource)
            headLayer.circleRadius      = NSExpression(forConstantValue: 6)
            headLayer.circleColor       = NSExpression(forConstantValue: color)
            headLayer.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
            headLayer.circleStrokeWidth = NSExpression(forConstantValue: 2)
            headLayer.circleOpacity     = NSExpression(forConstantValue: 0.9)
            style.addLayer(headLayer)
        }
    }

    // MARK: - Phase 2: Data update (called every frame)

    /// Called with `replayViewModel.visibleEntries` on every timer tick (~30fps).
    /// Updates source shapes in-place — no layer churn, no flicker.
    func updateData(visibleEntries: [LocationHistoryEntry]) {
        guard mapView?.style != nil else { return }
        guard !trackSources.isEmpty else { return }

        // Group visible entries by userId.
        var byUser: [String: [LocationHistoryEntry]] = [:]
        for entry in visibleEntries {
            byUser[entry.userId, default: []].append(entry)
        }

        for (userId, lineSource) in trackSources {
            let userEntries = byUser[userId] ?? []
            let sorted = userEntries.sorted { $0.recordedAt < $1.recordedAt }

            // Update the line shape.
            let coords: [CLLocationCoordinate2D] = sorted.map {
                CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
            }
            if coords.isEmpty {
                lineSource.shape = MLNShapeCollectionFeature(shapes: [])
            } else {
                lineSource.shape = MLNPolyline(coordinates: coords, count: UInt(coords.count))
            }

            // Update head marker: last visible point for this user.
            if let headSrc = headSources[userId] {
                if let last = sorted.last {
                    let point = MLNPointFeature()
                    point.coordinate = CLLocationCoordinate2D(latitude: last.lat, longitude: last.lng)
                    point.attributes = [
                        "username":   last.displayName.isEmpty ? last.username : last.displayName,
                        "deviceName": last.deviceName,
                    ]
                    headSrc.shape = MLNShapeCollectionFeature(shapes: [point])
                } else {
                    headSrc.shape = MLNShapeCollectionFeature(shapes: [])
                }
            }
        }
    }

    // MARK: - Cleanup

    /// Remove all layers and sources from the map style.
    func removeAll() {
        guard let mapView, let style = mapView.style else {
            trackSources.removeAll()
            headSources.removeAll()
            userColors.removeAll()
            colorIndex = 0
            return
        }

        for userId in trackSources.keys {
            if let layer = style.layer(withIdentifier: trackLineLayerId(userId)) {
                style.removeLayer(layer)
            }
            if let src = style.source(withIdentifier: trackLineSourceId(userId)) {
                style.removeSource(src)
            }
        }
        for userId in headSources.keys {
            if let layer = style.layer(withIdentifier: trackHeadLayerId(userId)) {
                style.removeLayer(layer)
            }
            if let src = style.source(withIdentifier: trackHeadSourceId(userId)) {
                style.removeSource(src)
            }
        }

        trackSources.removeAll()
        headSources.removeAll()
        userColors.removeAll()
        colorIndex = 0
    }

    // MARK: - Private helpers

    private func trackLineSourceId(_ userId: String) -> String { "history-line-\(userId)" }
    private func trackLineLayerId(_ userId: String)  -> String { "history-line-\(userId)-layer" }
    private func trackHeadSourceId(_ userId: String) -> String { "history-head-\(userId)" }
    private func trackHeadLayerId(_ userId: String)  -> String { "history-head-\(userId)-layer" }

    private func colorForUser(_ userId: String) -> UIColor {
        if let existing = userColors[userId] { return existing }
        let color = Self.trackColors[colorIndex % Self.trackColors.count]
        colorIndex += 1
        userColors[userId] = color
        return color
    }
}

// MARK: - UIColor Hex Helper

private extension UIColor {
    convenience init(hexString: String) {
        var hex = hexString.trimmingCharacters(in: .whitespacesAndNewlines)
        hex = hex.hasPrefix("#") ? String(hex.dropFirst()) : hex

        var rgb: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&rgb)

        self.init(
            red:   CGFloat((rgb >> 16) & 0xFF) / 255.0,
            green: CGFloat((rgb >> 8)  & 0xFF) / 255.0,
            blue:  CGFloat(rgb         & 0xFF) / 255.0,
            alpha: 1.0)
    }
}
