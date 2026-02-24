import Foundation
import MapLibre

/// Renders a GPX track on the MapLibre map.
///
/// GPX data is stored as GeoJSON in message metadata. This controller
/// adds a line layer for the track and point layers for start/end.
@MainActor
final class GPXOverlayController {

    private var mapView: MLNMapView?
    private var isRendered = false

    private static let sourceId = "gpx-overlay"
    private static let lineLayerId = "gpx-overlay-line"
    private static let pointLayerId = "gpx-overlay-points"

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    /// Render a GPX track from a message's GeoJSON data.
    func render(geojson: Data) {
        guard let mapView, let style = mapView.style else { return }

        // Remove existing
        removeOverlay()

        guard let shape = try? MLNShape(data: geojson, encoding: String.Encoding.utf8.rawValue)
        else { return }

        let source = MLNShapeSource(identifier: Self.sourceId, shape: shape, options: nil)
        style.addSource(source)

        // Line layer for the track
        let line = MLNLineStyleLayer(identifier: Self.lineLayerId, source: source)
        line.lineColor = NSExpression(forConstantValue: UIColor.systemPurple)
        line.lineWidth = NSExpression(forConstantValue: 3)
        line.lineOpacity = NSExpression(forConstantValue: 0.8)
        style.addLayer(line)

        // Point layer for waypoints/track endpoints
        let points = MLNCircleStyleLayer(identifier: Self.pointLayerId, source: source)
        points.circleRadius = NSExpression(forConstantValue: 5)
        points.circleColor = NSExpression(forConstantValue: UIColor.systemPurple)
        points.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
        points.circleStrokeWidth = NSExpression(forConstantValue: 2)
        style.addLayer(points)

        isRendered = true

        // Fit map to GPX bounds
        if let shapeSource = style.source(withIdentifier: Self.sourceId) as? MLNShapeSource,
           let features = shapeSource.shape
        {
            let bounds = features.coordinate
            mapView.setCenter(bounds, zoomLevel: max(mapView.zoomLevel, 12), animated: true)
        }
    }

    /// Remove the GPX overlay.
    func removeOverlay() {
        guard let mapView, let style = mapView.style, isRendered else { return }

        if let layer = style.layer(withIdentifier: Self.lineLayerId) {
            style.removeLayer(layer)
        }
        if let layer = style.layer(withIdentifier: Self.pointLayerId) {
            style.removeLayer(layer)
        }
        if let source = style.source(withIdentifier: Self.sourceId) {
            style.removeSource(source)
        }

        isRendered = false
    }
}
