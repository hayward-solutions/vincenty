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
/// - Styles shape annotations (MLNPolyline, MLNPolygon) for measure and draw tools
///
/// ## Double-tap gesture conflict
/// When a tool is active (`toolIsActive == true`) our custom double-tap recognizer
/// fires first.  The Coordinator conforms to `UIGestureRecognizerDelegate` and
/// implements `gestureRecognizerShouldBegin` so that:
///  - Tool active   → our recognizer fires; MapLibre's native zoom is blocked via
///                    `require(toFail:)` established at setup time.
///  - Tool inactive → `gestureRecognizerShouldBegin` returns `false` for ours, so
///                    only MapLibre's native double-tap fires (normal zoom).
struct MapContainerView: UIViewRepresentable {
    let settings: MapSettings?
    let onMapReady: (MLNMapView) -> Void
    let onCameraChanged: (Double, Double, Double) -> Void  // bearing, pitch, zoom
    var onUserDragBegan: (() -> Void)?
    var onTap: ((CLLocationCoordinate2D) -> Void)?
    var onDoubleTap: ((CLLocationCoordinate2D) -> Void)?

    /// Draw tool stroke color (hex string) — used to style "draw-line" annotations.
    var drawStrokeColor: String = "#3b82f6"
    /// Draw tool stroke width — used to style "draw-line" annotations.
    var drawStrokeWidth: Double = 2
    /// Draw tool fill color (hex string or "transparent") — used to style "draw-fill" annotations.
    var drawFillColor: String = "transparent"
    /// Whether a tool (measure or draw) is currently active.
    /// When true, our custom double-tap recognizer fires instead of MapLibre's zoom.
    var toolIsActive: Bool = false

    func makeUIView(context: Context) -> MapContainerUIView {
        let container = MapContainerUIView(frame: .zero)
        container.autoresizingMask = [.flexibleWidth, .flexibleHeight]
        let mapView = MLNMapView(frame: container.bounds)
        mapView.autoresizingMask = [.flexibleWidth, .flexibleHeight]
        container.mapView = mapView
        container.addSubview(mapView)
        mapView.delegate = context.coordinator
        mapView.logoView.isHidden = true
        mapView.attributionButton.isHidden = true

        // Hide the native compass — MapControlsView provides its own Reset North button
        mapView.compassView.compassVisibility = .hidden

        // Initial center/zoom from settings (avoids flash of default location)
        if let settings {
            mapView.setCenter(
                CLLocationCoordinate2D(latitude: settings.centerLat, longitude: settings.centerLng),
                zoomLevel: settings.zoom,
                animated: false)
        } else {
            mapView.setCenter(
                CLLocationCoordinate2D(latitude: 0, longitude: 0),
                zoomLevel: 2,
                animated: false)
        }

        // Single-tap for tool point placement
        let singleTap = UITapGestureRecognizer(
            target: context.coordinator, action: #selector(Coordinator.handleSingleTap(_:)))
        singleTap.numberOfTapsRequired = 1

        // Double-tap for tool finalization (line finish, etc.).
        // ToolDoubleTapRecognizer gates itself: it fails immediately when toolIsActive
        // is false, which unblocks MapLibre's native double-tap-to-zoom.
        let doubleTap = ToolDoubleTapRecognizer(
            target: context.coordinator, action: #selector(Coordinator.handleDoubleTap(_:)))
        doubleTap.numberOfTapsRequired = 2
        doubleTap.toolIsActive = toolIsActive

        // Single tap must wait for our double-tap to fail before firing
        singleTap.require(toFail: doubleTap)

        // Make MapLibre's native double-tap-to-zoom wait for our recognizer to fail.
        // When a tool is active our recognizer fires first; when inactive it fails
        // immediately so MapLibre's zoom proceeds unblocked.
        for recognizer in mapView.gestureRecognizers ?? [] {
            if let tap = recognizer as? UITapGestureRecognizer,
               tap.numberOfTapsRequired == 2
            {
                tap.require(toFail: doubleTap)
            }
        }

        mapView.addGestureRecognizer(singleTap)
        mapView.addGestureRecognizer(doubleTap)

        // Store reference so updateUIView can sync toolIsActive
        context.coordinator.customDoubleTap = doubleTap

        // Apply style
        applyStyle(to: mapView)

        return container
    }

    func updateUIView(_ container: MapContainerUIView, context: Context) {
        // Keep coordinator's parent reference current so delegate methods read
        // the latest draw style and toolIsActive on every SwiftUI re-render.
        context.coordinator.parent = self
        // Sync toolIsActive into the recognizer so it gates itself correctly.
        context.coordinator.customDoubleTap?.toolIsActive = toolIsActive
        // Ensure the map's frame tracks any SwiftUI-driven size changes
        // (orientation changes, split view, side panels opening/closing).
        container.setNeedsLayout()
        container.layoutIfNeeded()
    }

    /// Tell SwiftUI to give this representable the full proposed size.
    /// Without this, SwiftUI falls back to `systemLayoutSizeFitting`, which
    /// returns zero for a bare `UIView` container — leaving the map stuck at
    /// a tiny size on some layout passes (observed on iPad landscape).
    func sizeThatFits(
        _ proposal: ProposedViewSize,
        uiView: MapContainerUIView,
        context: Context
    ) -> CGSize? {
        CGSize(
            width: proposal.width ?? UIView.layoutFittingExpandedSize.width,
            height: proposal.height ?? UIView.layoutFittingExpandedSize.height)
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    // MARK: - Style Building

    /// Builds and sets the MapLibre style, mirroring web's `buildStyle`.
    private func applyStyle(to mapView: MLNMapView) {
        guard let settings else {
            let defaultURL = URL(string: "https://demotiles.maplibre.org/style.json")!
            mapView.styleURL = defaultURL
            return
        }

        // If the server provides a full style JSON dict, serialize and use it
        if let styleJSON = settings.styleJson, !styleJSON.isEmpty {
            let rawDict = styleJSON.mapValues { $0.value }
            if let data = try? JSONSerialization.data(withJSONObject: rawDict) {
                let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent(
                    "mapstyle-\(UUID().uuidString).json")
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
                    "mapstyle-\(UUID().uuidString).json")
                try? data.write(to: tempURL)
                mapView.styleURL = tempURL
            }
        } else {
            mapView.styleURL = URL(string: "https://demotiles.maplibre.org/style.json")!
        }
    }

    /// Appends API keys to tile URLs for known providers, mirroring the web
    /// client's `transformRequest` callback in `map-view.tsx`.
    func transformURL(_ url: String, settings: MapSettings) -> String {
        let mapboxToken = settings.mapboxAccessToken ?? ""
        if !mapboxToken.isEmpty,
           url.contains("mapbox.com") || url.contains("tiles.mapbox.com")
        {
            let separator = url.contains("?") ? "&" : "?"
            return "\(url)\(separator)access_token=\(mapboxToken)"
        }
        let googleKey = settings.googleMapsApiKey ?? ""
        if !googleKey.isEmpty, url.contains("googleapis.com") {
            let separator = url.contains("?") ? "&" : "?"
            return "\(url)\(separator)key=\(googleKey)"
        }
        return url
    }

    private func buildRasterStyle(tileURL: String, settings: MapSettings) -> String {
        let center = [settings.centerLng, settings.centerLat]
        let zoom = Int(settings.zoom)

        let authenticatedURL = transformURL(tileURL, settings: settings)

        let tileURLJSON: String
        if authenticatedURL.contains("{s}") {
            let subdomains = ["a", "b", "c"]
            let tiles = subdomains.map { s in
                "\"\(authenticatedURL.replacingOccurrences(of: "{s}", with: s))\""
            }
            tileURLJSON = "[\(tiles.joined(separator: ","))]"
        } else {
            tileURLJSON = "[\"\(authenticatedURL)\"]"
        }

        return """
            {
              "version": 8,
              "name": "Vincenty Raster",
              "center": [\(center[0]), \(center[1])],
              "zoom": \(zoom),
              "sources": {
                "raster-tiles": {
                  "type": "raster",
                  "tiles": \(tileURLJSON),
                  "tileSize": 256,
                  "maxzoom": \(settings.maxZoom),
                  "attribution": ""
                }
              },
              "layers": [
                {
                  "id": "raster-layer",
                  "type": "raster",
                  "source": "raster-tiles",
                  "minzoom": \(settings.minZoom),
                  "maxzoom": \(settings.maxZoom)
                }
              ]
            }
            """
    }

    // MARK: - Coordinator

    @MainActor
    final class Coordinator: NSObject, @preconcurrency MLNMapViewDelegate {
        var parent: MapContainerView
        private var didNotifyReady = false

        /// Reference to our custom double-tap recognizer, set in makeUIView.
        /// Cast to ToolDoubleTapRecognizer to update toolIsActive in updateUIView.
        var customDoubleTap: ToolDoubleTapRecognizer?

        init(parent: MapContainerView) {
            self.parent = parent
        }

        // MARK: - Map Style Loaded

        func mapView(_ mapView: MLNMapView, didFinishLoading style: MLNStyle) {
            if let settings = parent.settings, !settings.terrainUrl.isEmpty {
                addTerrainSource(to: style, terrainURL: settings.terrainUrl, settings: settings)
            }

            if let settings = parent.settings {
                mapView.setCenter(
                    CLLocationCoordinate2D(
                        latitude: settings.centerLat, longitude: settings.centerLng),
                    animated: false)
                mapView.setZoomLevel(settings.zoom, animated: false)
            }

            if !didNotifyReady {
                didNotifyReady = true
                parent.onMapReady(mapView)
            }
        }

        // MARK: - Camera

        func mapView(
            _ mapView: MLNMapView,
            regionWillChangeWith reason: MLNCameraChangeReason,
            animated: Bool
        ) {
            let gestureFlags: MLNCameraChangeReason = [
                .gesturePan, .gesturePinch, .gestureRotate,
                .gestureZoomIn, .gestureZoomOut, .gestureOneFingerZoom, .gestureTilt,
            ]
            if !reason.intersection(gestureFlags).isEmpty {
                parent.onUserDragBegan?()
            }
        }

        func mapView(
            _ mapView: MLNMapView,
            regionDidChangeWith reason: MLNCameraChangeReason,
            animated: Bool
        ) {
            parent.onCameraChanged(
                mapView.direction,
                mapView.camera.pitch,
                mapView.zoomLevel)
        }

        // MARK: - Annotation Views

        /// Return a custom view for known annotation types; nil → default pin.
        ///
        ///   "You"            → SelfAnnotationView (16pt blue dot)
        ///   "measure-point"  → SmallDotAnnotationView (8pt blue)
        ///   "measure-center" → SmallDotAnnotationView (10pt blue)
        ///   "draw-point"     → SmallDotAnnotationView (8pt, draw stroke color)
        func mapView(_ mapView: MLNMapView, viewFor annotation: MLNAnnotation) -> MLNAnnotationView? {
            guard let point = annotation as? MLNPointAnnotation else { return nil }

            // Location markers — identified by type, not title, so any display name works.
            if let locAnnotation = point as? LocationAnnotation {
                let reuseId = "location-marker"
                let view = (mapView.dequeueReusableAnnotationView(withIdentifier: reuseId)
                    as? LocationAnnotationView) ?? LocationAnnotationView(reuseIdentifier: reuseId)
                view.configure(color: locAnnotation.color)
                return view
            }

            switch point.title {
            case "You":
                let reuseId = "self-marker"
                if let view = mapView.dequeueReusableAnnotationView(withIdentifier: reuseId) {
                    return view
                }
                return SelfAnnotationView(reuseIdentifier: reuseId)

            case "measure-point":
                let reuseId = "measure-point"
                if let view = mapView.dequeueReusableAnnotationView(withIdentifier: reuseId) {
                    return view
                }
                return SmallDotAnnotationView(reuseIdentifier: reuseId, size: 8, color: .systemBlue)

            case "measure-center":
                let reuseId = "measure-center"
                if let view = mapView.dequeueReusableAnnotationView(withIdentifier: reuseId) {
                    return view
                }
                return SmallDotAnnotationView(reuseIdentifier: reuseId, size: 10, color: .systemBlue)

            case "draw-point":
                // Don't dequeue — color changes with drawStrokeColor
                return SmallDotAnnotationView(
                    reuseIdentifier: "draw-point",
                    size: 8,
                    color: UIColor(hex: parent.drawStrokeColor))

            default:
                return nil
            }
        }

        // MARK: - Shape Annotation Styling

        /// Stroke / outline color for MLNPolyline and MLNPolygon border.
        func mapView(
            _ mapView: MLNMapView, strokeColorForShapeAnnotation annotation: MLNShape
        ) -> UIColor {
            switch annotation.title {
            case "measure-line":    return .systemBlue
            case "measure-outline": return .systemBlue
            case "measure-radius":  return UIColor.systemBlue.withAlphaComponent(0.7)
            case "measure-fill":    return .systemBlue
            case "draw-line":       return UIColor(hex: parent.drawStrokeColor)
            case "draw-fill":       return UIColor(hex: parent.drawStrokeColor)
            default:                return .systemBlue
            }
        }

        /// Line width for MLNPolyline annotations.
        func mapView(
            _ mapView: MLNMapView, lineWidthForPolylineAnnotation annotation: MLNPolyline
        ) -> CGFloat {
            switch annotation.title {
            case "measure-line":    return 3
            case "measure-outline": return 2
            case "measure-radius":  return 2
            case "draw-line":       return CGFloat(parent.drawStrokeWidth)
            default:                return 2
            }
        }

        /// Fill color for MLNPolygon annotations.
        func mapView(
            _ mapView: MLNMapView, fillColorForPolygonAnnotation annotation: MLNPolygon
        ) -> UIColor {
            switch annotation.title {
            case "measure-fill":
                return UIColor.systemBlue.withAlphaComponent(0.15)
            case "draw-fill":
                let hex = parent.drawFillColor
                if hex == "transparent" { return .clear }
                return UIColor(hex: hex).withAlphaComponent(0.25)
            default:
                return UIColor.systemBlue.withAlphaComponent(0.15)
            }
        }

        /// Overall opacity for shape annotations.
        func mapView(
            _ mapView: MLNMapView, alphaForShapeAnnotation annotation: MLNShape
        ) -> CGFloat {
            return 1.0
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
            let authenticatedURL = parent.transformURL(terrainURL, settings: settings)
            let encoding = settings.terrainEncoding.isEmpty ? "terrarium" : settings.terrainEncoding
            let tileSize = encoding == "mapbox" ? 512 : 256
            let options: [MLNTileSourceOption: Any] = [.tileSize: NSNumber(value: tileSize)]
            let source = MLNRasterDEMSource(
                identifier: "terrain-dem",
                tileURLTemplates: [authenticatedURL],
                options: options)
            style.addSource(source)
        }
    }
}

// MARK: - Map Container UIView

/// A thin `UIView` wrapper that hosts `MLNMapView` as a subview and forces
/// its frame to match `bounds` on every layout pass.
///
/// This works around an iPad landscape bug where MapLibre's GL/Metal drawable
/// surface desyncs from the SwiftUI-provided bounds after orientation changes
/// (map only renders in the top-left corner). Overriding `layoutSubviews`
/// guarantees the map's frame tracks the container's bounds, and MapLibre
/// reshapes its drawable accordingly.
final class MapContainerUIView: UIView {
    var mapView: MLNMapView?

    override func layoutSubviews() {
        super.layoutSubviews()
        guard let mapView else { return }
        mapView.frame = bounds
        mapView.setNeedsLayout()
        mapView.layoutIfNeeded()
    }

    /// Force a layout + Metal drawable resync when the view is reattached to a
    /// window. In a `TabView`, switching tabs detaches this view from the
    /// window; on return, bounds are unchanged so `layoutSubviews` alone won't
    /// re-trigger MapLibre's internal resize of its `CAMetalLayer`. Nudging
    /// the map view's layer directly forces the drawable size to refresh.
    override func didMoveToWindow() {
        super.didMoveToWindow()
        guard window != nil, let mapView else { return }
        mapView.frame = bounds
        mapView.setNeedsLayout()
        mapView.layoutIfNeeded()
        // Kick the Metal layer: contentsScale assignment forces a drawable
        // size recompute even when bounds are unchanged.
        let scale = mapView.layer.contentsScale
        mapView.layer.contentsScale = scale
        mapView.setNeedsDisplay()
    }
}

// MARK: - Self Annotation View

/// Blue dot for the current user's position. Triggered by annotation title "You".
private final class SelfAnnotationView: MLNAnnotationView {

    private static let size: CGFloat = 16

    override init(reuseIdentifier: String?) {
        super.init(reuseIdentifier: reuseIdentifier)
        build()
    }

    required init?(coder: NSCoder) { nil }

    private func build() {
        let s = Self.size
        bounds = CGRect(x: 0, y: 0, width: s, height: s)
        backgroundColor = UIColor.systemBlue
        layer.cornerRadius = s / 2
        layer.borderColor = UIColor.white.cgColor
        layer.borderWidth = 2
        layer.shadowColor = UIColor.black.cgColor
        layer.shadowOpacity = 0.25
        layer.shadowOffset = CGSize(width: 0, height: 1)
        layer.shadowRadius = 2
        centerOffset = CGVector(dx: 0, dy: 0)
    }
}

// MARK: - Small Dot Annotation View

/// Small circular dot for measure/draw tool vertex markers.
private final class SmallDotAnnotationView: MLNAnnotationView {

    override init(reuseIdentifier: String?) {
        super.init(reuseIdentifier: reuseIdentifier)
    }

    convenience init(reuseIdentifier: String?, size: CGFloat, color: UIColor) {
        self.init(reuseIdentifier: reuseIdentifier)
        bounds = CGRect(x: 0, y: 0, width: size, height: size)
        backgroundColor = color
        layer.cornerRadius = size / 2
        layer.borderColor = UIColor.white.cgColor
        layer.borderWidth = 1.5
        layer.shadowColor = UIColor.black.cgColor
        layer.shadowOpacity = 0.2
        layer.shadowOffset = CGSize(width: 0, height: 1)
        layer.shadowRadius = 1
        centerOffset = CGVector(dx: 0, dy: 0)
    }

    required init?(coder: NSCoder) { nil }
}

// MARK: - Location Annotation

/// Point annotation subclass for other users' device positions.
/// Carries a `color` property so the delegate can render a per-group/per-user
/// coloured marker. Using a distinct type allows the delegate to route it
/// independently of the title-based tool annotations.
final class LocationAnnotation: MLNPointAnnotation {
    var color: UIColor = .systemBlue
}

// MARK: - Location Annotation View

/// Coloured circle for another user's device position.
/// 14 pt — slightly smaller than the 16 pt self-marker to give visual hierarchy.
private final class LocationAnnotationView: MLNAnnotationView {

    private static let size: CGFloat = 14

    override init(reuseIdentifier: String?) {
        super.init(reuseIdentifier: reuseIdentifier)
        build()
    }

    required init?(coder: NSCoder) { nil }

    /// Apply a colour to this view. Called after dequeue so reused views
    /// get the correct colour for the annotation they are now representing.
    func configure(color: UIColor) {
        backgroundColor = color
    }

    private func build() {
        let s = Self.size
        bounds = CGRect(x: 0, y: 0, width: s, height: s)
        backgroundColor = .systemBlue
        layer.cornerRadius = s / 2
        layer.borderColor = UIColor.white.cgColor
        layer.borderWidth = 2
        layer.shadowColor = UIColor.black.cgColor
        layer.shadowOpacity = 0.25
        layer.shadowOffset = CGSize(width: 0, height: 1)
        layer.shadowRadius = 2
        centerOffset = CGVector(dx: 0, dy: 0)
    }
}

// MARK: - Tool Double-Tap Recognizer

/// A UITapGestureRecognizer that gates itself on `toolIsActive`.
///
/// When `toolIsActive` is false the recognizer immediately transitions to `.failed`
/// in `touchesBegan`.  Because MapLibre's native double-tap-to-zoom was set up with
/// `require(toFail: this)`, its failure unblocks MapLibre's zoom — so normal
/// pinch-to-zoom double-tap still works when no tool is active.
///
/// When `toolIsActive` is true the recognizer proceeds normally and MapLibre's zoom
/// is suppressed for that gesture.
final class ToolDoubleTapRecognizer: UITapGestureRecognizer {
    var toolIsActive: Bool = false

    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent) {
        if toolIsActive {
            super.touchesBegan(touches, with: event)
        } else {
            state = .failed
        }
    }
}
