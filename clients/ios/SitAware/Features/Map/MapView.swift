import MapLibre
import SwiftUI

/// SwiftUI wrapper for `MLNMapView` (MapLibre Native).
///
/// Mirrors the web client's `map-view.tsx`:
/// - Builds the map style from server `MapSettings` (style_json or raster tiles)
/// - Adds terrain DEM source if configured
/// - Reports `isReady` after the style loads
/// - Exposes the underlying `MLNMapView` via `onMapReady` callback
/// - Delegates camera changes back to the view model
/// - Forwards tap and double-tap gestures for draw/measure tool interaction
struct MapContainerView: UIViewRepresentable {
    let settings: MapSettings?
    let onMapReady: (MLNMapView) -> Void
    let onCameraChanged: (Double, Double, Double) -> Void  // bearing, pitch, zoom
    var onTap: ((CLLocationCoordinate2D) -> Void)?
    var onDoubleTap: ((CLLocationCoordinate2D) -> Void)?

    func makeUIView(context: Context) -> MLNMapView {
        let mapView = MLNMapView(frame: .zero)
        mapView.delegate = context.coordinator
        mapView.logoView.isHidden = true
        mapView.attributionButton.isHidden = true

        // Enable compass but let us control rotation reporting
        mapView.compassView.compassVisibility = .visible

        // Default location
        mapView.setCenter(
            CLLocationCoordinate2D(latitude: -33.8688, longitude: 151.2093),
            zoomLevel: 10,
            animated: false)

        // Add tap gesture recognizer for draw/measure tool interaction
        let singleTap = UITapGestureRecognizer(
            target: context.coordinator, action: #selector(Coordinator.handleSingleTap(_:)))
        singleTap.numberOfTapsRequired = 1
        let doubleTap = UITapGestureRecognizer(
            target: context.coordinator, action: #selector(Coordinator.handleDoubleTap(_:)))
        doubleTap.numberOfTapsRequired = 2
        // Single tap waits for double-tap to fail before firing
        singleTap.require(toFail: doubleTap)
        mapView.addGestureRecognizer(singleTap)
        mapView.addGestureRecognizer(doubleTap)

        // Apply style
        applyStyle(to: mapView)

        return mapView
    }

    func updateUIView(_ mapView: MLNMapView, context: Context) {
        // Style changes are handled via the coordinator's didFinishLoading callback
        // Only re-apply if settings changed (detected by coordinator)
        context.coordinator.parent = self
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    // MARK: - Style Building

    /// Builds and sets the MapLibre style, mirroring web's `buildStyle`.
    private func applyStyle(to mapView: MLNMapView) {
        guard let settings else {
            // Default OSM raster style
            let defaultURL = URL(string: "https://demotiles.maplibre.org/style.json")!
            mapView.styleURL = defaultURL
            return
        }

        // If the server provides a full style JSON dict, serialize and use it
        if let styleJSON = settings.styleJson, !styleJSON.isEmpty {
            // Convert [String: AnyCodable] to JSON Data
            let rawDict = styleJSON.mapValues { $0.value }
            if let data = try? JSONSerialization.data(withJSONObject: rawDict) {
                let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent(
                    "mapstyle.json")
                try? data.write(to: tempURL)
                mapView.styleURL = tempURL
                return
            }
        }

        // Build raster tile style from settings
        if !settings.tileUrl.isEmpty {
            let styleJSON = buildRasterStyle(tileURL: settings.tileUrl, settings: settings)
            if let data = styleJSON.data(using: .utf8) {
                let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent(
                    "mapstyle.json")
                try? data.write(to: tempURL)
                mapView.styleURL = tempURL
            }
        } else {
            // Fallback to demo tiles
            mapView.styleURL = URL(string: "https://demotiles.maplibre.org/style.json")!
        }
    }

    /// Build a minimal raster tile style JSON, matching the web's inline style builder.
    private func buildRasterStyle(tileURL: String, settings: MapSettings) -> String {
        let center = [settings.centerLng, settings.centerLat]
        let zoom = Int(settings.zoom)

        // Detect if tile URL contains {s} subdomains
        let tileURLJSON: String
        if tileURL.contains("{s}") {
            // Expand subdomains
            let subdomains = ["a", "b", "c"]
            let tiles = subdomains.map { s in
                "\"\(tileURL.replacingOccurrences(of: "{s}", with: s))\""
            }
            tileURLJSON = "[\(tiles.joined(separator: ","))]"
        } else {
            tileURLJSON = "[\"\(tileURL)\"]"
        }

        return """
            {
              "version": 8,
              "name": "SitAware Raster",
              "center": [\(center[0]), \(center[1])],
              "zoom": \(zoom),
              "sources": {
                "raster-tiles": {
                  "type": "raster",
                  "tiles": \(tileURLJSON),
                  "tileSize": 256,
                  "attribution": ""
                }
              },
              "layers": [
                {
                  "id": "raster-layer",
                  "type": "raster",
                  "source": "raster-tiles",
                  "minzoom": 0,
                  "maxzoom": 22
                }
              ]
            }
            """
    }

    // MARK: - Coordinator

    final class Coordinator: NSObject, MLNMapViewDelegate {
        var parent: MapContainerView
        private var didNotifyReady = false

        init(parent: MapContainerView) {
            self.parent = parent
        }

        func mapView(_ mapView: MLNMapView, didFinishLoading style: MLNStyle) {
            // Add terrain DEM source if configured
            if let settings = parent.settings, !settings.terrainUrl.isEmpty {
                addTerrainSource(to: style, terrainURL: settings.terrainUrl, settings: settings)
            }

            // Apply default center/zoom from settings
            if let settings = parent.settings {
                mapView.setCenter(
                    CLLocationCoordinate2D(latitude: settings.centerLat, longitude: settings.centerLng),
                    animated: false)
                mapView.setZoomLevel(settings.zoom, animated: false)
            }

            if !didNotifyReady {
                didNotifyReady = true
                parent.onMapReady(mapView)
            }
        }

        func mapView(_ mapView: MLNMapView, regionDidChangeAnimated _: Bool) {
            parent.onCameraChanged(
                mapView.direction,  // bearing (0-360)
                mapView.camera.pitch,
                mapView.zoomLevel)
        }

        // MARK: - Tap Gestures

        @objc func handleSingleTap(_ gesture: UITapGestureRecognizer) {
            guard gesture.state == .ended,
                  let mapView = gesture.view as? MLNMapView
            else { return }
            let point = gesture.location(in: mapView)
            let coordinate = mapView.convert(point, toCoordinateFrom: mapView)
            parent.onTap?(coordinate)
        }

        @objc func handleDoubleTap(_ gesture: UITapGestureRecognizer) {
            guard gesture.state == .ended,
                  let mapView = gesture.view as? MLNMapView
            else { return }
            let point = gesture.location(in: mapView)
            let coordinate = mapView.convert(point, toCoordinateFrom: mapView)
            parent.onDoubleTap?(coordinate)
        }

        // MARK: - Terrain

        private func addTerrainSource(
            to style: MLNStyle, terrainURL: String, settings: MapSettings
        ) {
            let encoding = settings.terrainEncoding ?? "terrarium"
            let tileSize = encoding == "mapbox" ? 512 : 256

            // Check if it's a TileJSON endpoint or direct tile URL
            if terrainURL.hasSuffix(".json") {
                // TileJSON URL — use URL property
                let options: [MLNTileSourceOption: Any] = [
                    .tileSize: NSNumber(value: tileSize)
                ]
                let source = MLNRasterDEMSource(
                    identifier: "terrain-dem",
                    tileURLTemplates: [terrainURL],
                    options: options)
                style.addSource(source)
            } else {
                let options: [MLNTileSourceOption: Any] = [
                    .tileSize: NSNumber(value: tileSize)
                ]
                let source = MLNRasterDEMSource(
                    identifier: "terrain-dem",
                    tileURLTemplates: [terrainURL],
                    options: options)
                style.addSource(source)
            }
        }
    }
}
