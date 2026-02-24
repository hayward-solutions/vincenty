import Foundation
import MapLibre

/// Renders saved GeoJSON drawings on the MapLibre map as persistent overlays.
///
/// Mirrors the web client's `drawing-overlay.tsx`:
/// - Each drawing gets its own source + fill/outline/line/point layer set
/// - Diff-based: only adds/removes/updates drawings that changed
/// - Data-driven styling from per-feature `stroke`, `fill`, `strokeWidth` properties
@MainActor
final class DrawingOverlayController {

    private var mapView: MLNMapView?

    /// Tracks rendered drawing IDs → their `updatedAt` value for diffing.
    private var rendered: [String: String] = [:]

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    /// Sync visible drawings onto the map. Call whenever visibility or drawing data changes.
    func update(drawings: [DrawingResponse]) {
        guard let mapView, let style = mapView.style else { return }

        let currentIds = Set(drawings.map(\.id))

        // Remove stale drawings
        let staleIds = Set(rendered.keys).subtracting(currentIds)
        for id in staleIds {
            removeDrawing(id, from: style)
            rendered.removeValue(forKey: id)
        }

        // Add or update drawings
        for drawing in drawings {
            if let existingUpdatedAt = rendered[drawing.id] {
                if existingUpdatedAt != drawing.updatedAt {
                    // Updated — refresh data
                    updateDrawingData(drawing, in: style)
                    rendered[drawing.id] = drawing.updatedAt
                }
                // Otherwise: unchanged, skip
            } else {
                // New drawing
                addDrawing(drawing, to: style)
                rendered[drawing.id] = drawing.updatedAt
            }
        }
    }

    /// Remove all rendered drawings.
    func removeAll() {
        guard let mapView, let style = mapView.style else { return }
        for id in rendered.keys {
            removeDrawing(id, from: style)
        }
        rendered.removeAll()
    }

    // MARK: - Private

    private func addDrawing(_ drawing: DrawingResponse, to style: MLNStyle) {
        guard let geoJSONData = encodeGeoJSON(drawing.geojson) else { return }
        guard let shape = try? MLNShape(data: geoJSONData, encoding: String.Encoding.utf8.rawValue)
        else { return }

        let sourceId = "drawing-\(drawing.id)"
        let source = MLNShapeSource(identifier: sourceId, shape: shape, options: nil)
        style.addSource(source)

        // Fill layer (polygons)
        let fill = MLNFillStyleLayer(identifier: "\(sourceId)-fill", source: source)
        fill.fillColor = NSExpression(
            forMLNJSONObject: ["coalesce", ["get", "fill"], "#3b82f6"])
        fill.fillOpacity = NSExpression(forConstantValue: 0.2)
        style.addLayer(fill)

        // Outline layer (polygon borders)
        let outline = MLNLineStyleLayer(identifier: "\(sourceId)-outline", source: source)
        outline.lineColor = NSExpression(
            forMLNJSONObject: ["coalesce", ["get", "stroke"], "#3b82f6"])
        outline.lineWidth = NSExpression(
            forMLNJSONObject: ["coalesce", ["get", "strokeWidth"], 2])
        outline.lineOpacity = NSExpression(forConstantValue: 0.9)
        style.addLayer(outline)

        // Line layer (linestrings)
        let line = MLNLineStyleLayer(identifier: "\(sourceId)-line", source: source)
        line.lineColor = outline.lineColor
        line.lineWidth = outline.lineWidth
        line.lineOpacity = NSExpression(forConstantValue: 0.9)
        style.addLayer(line)

        // Point layer (vertices)
        let point = MLNCircleStyleLayer(identifier: "\(sourceId)-point", source: source)
        point.circleRadius = NSExpression(forConstantValue: 4)
        point.circleColor = NSExpression(
            forMLNJSONObject: ["coalesce", ["get", "stroke"], "#3b82f6"])
        point.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
        point.circleStrokeWidth = NSExpression(forConstantValue: 1.5)
        style.addLayer(point)
    }

    private func updateDrawingData(_ drawing: DrawingResponse, in style: MLNStyle) {
        let sourceId = "drawing-\(drawing.id)"
        guard let source = style.source(withIdentifier: sourceId) as? MLNShapeSource,
              let geoJSONData = encodeGeoJSON(drawing.geojson),
              let shape = try? MLNShape(
                data: geoJSONData, encoding: String.Encoding.utf8.rawValue)
        else { return }

        source.shape = shape
    }

    private func removeDrawing(_ id: String, from style: MLNStyle) {
        let sourceId = "drawing-\(id)"
        let layerIds = ["\(sourceId)-fill", "\(sourceId)-outline", "\(sourceId)-line",
                        "\(sourceId)-point"]

        for layerId in layerIds {
            if let layer = style.layer(withIdentifier: layerId) {
                style.removeLayer(layer)
            }
        }
        if let source = style.source(withIdentifier: sourceId) {
            style.removeSource(source)
        }
    }

    private func encodeGeoJSON(_ featureCollection: GeoJSONFeatureCollection) -> Data? {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        return try? encoder.encode(featureCollection)
    }
}
