import SwiftUI
import CoreLocation

/// The main map screen — full-viewport MapLibre map with overlaid controls.
///
/// Mirrors the web client's `app/(app)/map/page.tsx` layout:
/// - MapContainerView as the base layer
/// - MapToolbarView top-left
/// - MapControlsView top-right
/// - MapFilterPanel below toolbar (when active)
/// - MeasurePanelView below toolbar (when measure active)
/// - DrawPanelView below toolbar (when draw active)
/// - ReplayControlsView at bottom (when replay active)
/// - Loading spinner while settings load
///
/// Tap and double-tap gestures are forwarded to the active tool controller
/// (measure or draw), providing point-by-point interaction on the map.
struct MapScreen: View {
    @Environment(AuthManager.self) private var auth
    @Environment(WebSocketService.self) private var webSocket
    @Environment(LocationSharingManager.self) private var locationSharing
    @Environment(DeviceManager.self) private var deviceManager

    @State private var viewModel = MapViewModel()
    @State private var locationMarkers = LocationMarkersController()
    @State private var selfMarker = SelfMarkerController()

    // MARK: - Measure Tool State

    @State private var measureController = MeasureToolController()
    @State private var measureMode: MeasureMode = .line
    @State private var measureResult = MeasureResult.empty

    // MARK: - Draw Tool State

    @State private var drawController = DrawToolController()
    @State private var drawingsViewModel = DrawingsViewModel()

    // MARK: - Replay State

    @State private var replayViewModel = ReplayViewModel()

    var body: some View {
        ZStack {
            if viewModel.isLoadingSettings {
                // Loading state while fetching map config
                LoadingStateView(message: "Loading map...", style: .fullScreen)
            } else {
                // Map base layer
                mapContainerView

                // Overlaid controls
                mapOverlays
            }
        }
        .task {
            await viewModel.loadInitialData()
        }
        .task {
            viewModel.subscribeToLocations(webSocket: webSocket)
        }
        .task {
            // Start location sharing once map appears
            if let deviceId = deviceManager.deviceId {
                locationSharing.startSharing(webSocket: webSocket, deviceId: deviceId)
            }
        }
        .onDisappear {
            viewModel.unsubscribe()
            measureController.deactivate()
            drawController.deactivate()
        }
        .onChange(of: viewModel.displayLocations) { _, locations in
            locationMarkers.update(
                locations: locations,
                currentDeviceId: deviceManager.deviceId,
                groups: viewModel.groups)
        }
        .onChange(of: locationSharing.currentPosition?.lat) { _, _ in
            updateSelfPosition()
        }
        .onChange(of: locationSharing.currentPosition?.lng) { _, _ in
            updateSelfPosition()
        }
        // Measure tool lifecycle
        .onChange(of: viewModel.showMeasurePanel) { _, isActive in
            if isActive {
                measureController.activate(mode: measureMode)
                measureController.onMeasurementsChange = { result in
                    measureResult = result
                }
            } else {
                measureController.deactivate()
                measureResult = .empty
            }
        }
        // Draw tool lifecycle
        .onChange(of: viewModel.showDrawPanel) { _, isActive in
            if isActive {
                drawController.activate(mode: drawingsViewModel.drawMode, style: drawingsViewModel.drawStyle)
                drawController.onShapeComplete = { feature, mode in
                    drawingsViewModel.completedShapes.append(CompletedShape(feature: feature, shapeType: mode))
                }
                // Restore any shapes completed before the panel was re-opened
                drawController.updateCompletedShapes(drawingsViewModel.completedShapes)
            } else {
                drawController.deactivate()
            }
        }
        // Sync completed shapes to map whenever the list changes
        .onChange(of: drawingsViewModel.completedShapes.count) { _, _ in
            if drawController.isActive {
                drawController.updateCompletedShapes(drawingsViewModel.completedShapes)
            }
        }
        // Sync draw mode/style changes to controller
        .onChange(of: drawingsViewModel.drawMode) { _, newMode in
            if drawController.isActive {
                drawController.updateMode(newMode)
            }
        }
        .onChange(of: drawingsViewModel.drawStyle) { _, newStyle in
            if drawController.isActive {
                drawController.updateStyle(newStyle)
            }
        }
    }

    // MARK: - Location Updates

    /// Sync the latest GPS position to the view model and self-marker.
    private func updateSelfPosition() {
        viewModel.selfPosition = locationSharing.currentPosition
        selfMarker.update(
            position: viewModel.showSelf ? viewModel.selfPosition : nil,
            autoCenter: true)
        viewModel.updateTrackingIfNeeded()
    }

    // MARK: - Tap Handling

    /// Route map taps to the active tool controller.
    private func handleMapTap(_ coordinate: CLLocationCoordinate2D) {
        if measureController.isActive {
            measureController.handleTap(at: coordinate)
        } else if drawController.isActive {
            drawController.handleTap(at: coordinate)
        }
    }

    /// Route map double-taps to the active tool controller.
    private func handleMapDoubleTap(_ coordinate: CLLocationCoordinate2D) {
        if measureController.isActive {
            measureController.handleDoubleTap(at: coordinate)
        } else if drawController.isActive {
            drawController.handleDoubleTap(at: coordinate)
        }
    }

    // MARK: - Map Container

    /// Extracted to a separate property to keep `body` short enough for the Swift type-checker.
    @ViewBuilder
    private var mapContainerView: some View {
        MapContainerView(
            settings: viewModel.mapSettings,
            onMapReady: { mapView in
                viewModel.onMapReady(mapView)
                locationMarkers.attach(to: mapView)
                selfMarker.attach(to: mapView)
                measureController.attach(to: mapView)
                drawController.attach(to: mapView)
                // Re-apply current position immediately — handles the case where
                // the map was recreated and onChange won't fire (value unchanged).
                updateSelfPosition()
                // Re-apply other device locations — handles the common race where
                // the WS location_snapshot arrives before the map style finishes
                // loading, causing the onChange to no-op (mapView was nil).
                locationMarkers.update(
                    locations: viewModel.displayLocations,
                    currentDeviceId: deviceManager.deviceId,
                    groups: viewModel.groups
                )
            },
            onCameraChanged: { bearing, pitch, zoom in
                viewModel.onCameraChanged(bearing: bearing, pitch: pitch, zoom: zoom)
            },
            onUserDragBegan: {
                viewModel.onUserDragBegan()
            },
            onTap: handleMapTap,
            onDoubleTap: handleMapDoubleTap,
            drawStrokeColor: drawingsViewModel.drawStyle.stroke,
            drawStrokeWidth: drawingsViewModel.drawStyle.strokeWidth,
            drawFillColor: drawingsViewModel.drawStyle.fill,
            toolIsActive: viewModel.showMeasurePanel || viewModel.showDrawPanel
        )
        .ignoresSafeArea()
    }

    // MARK: - Overlays

    @ViewBuilder
    private var mapOverlays: some View {
        // Top-left: toolbar + panels
        VStack(alignment: .leading, spacing: 8) {
            MapToolbarView(viewModel: viewModel)

            if viewModel.showFilterPanel {
                MapFilterPanel(viewModel: viewModel)
                    .transition(.opacity.combined(with: .move(edge: .top)))
            }

            if viewModel.showMeasurePanel {
                MeasurePanelView(
                    mode: measureMode,
                    measurements: measureResult,
                    onModeChange: { newMode in
                        measureMode = newMode
                        measureController.updateMode(newMode)
                        measureResult = .empty
                    },
                    onClear: {
                        measureController.clear()
                        measureResult = .empty
                    },
                    onClose: {
                        viewModel.toggleMeasure()
                    }
                )
                .transition(.opacity.combined(with: .move(edge: .top)))
            }

            if viewModel.showDrawPanel {
                DrawPanelView(
                    viewModel: drawingsViewModel,
                    groups: viewModel.groups,
                    onClose: {
                        viewModel.toggleDraw()
                    }
                )
                .transition(.opacity.combined(with: .move(edge: .top)))
            }

            Spacer()
        }
        .padding(.leading, 12)
        .padding(.top, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .animation(.easeInOut(duration: 0.2), value: viewModel.showFilterPanel)
        .animation(.easeInOut(duration: 0.2), value: viewModel.showMeasurePanel)
        .animation(.easeInOut(duration: 0.2), value: viewModel.showDrawPanel)

        // Top-right: map controls
        VStack {
            MapControlsView(viewModel: viewModel)
            Spacer()
        }
        .padding(.trailing, 12)
        .padding(.top, 12)
        .frame(maxWidth: .infinity, alignment: .trailing)

        // Top-center: WebSocket connection status (hidden when connected)
        if webSocket.connectionState != .connected {
            VStack {
                StatusBanner(
                    icon: webSocket.connectionState == .connecting ? nil : "wifi.slash",
                    message: webSocket.connectionState == .connecting
                        ? "Connecting..." : "Disconnected",
                    color: webSocket.connectionState == .connecting ? .orange : .red,
                    showSpinner: webSocket.connectionState == .connecting
                )
                Spacer()
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 12)
            .transition(.move(edge: .top).combined(with: .opacity))
            .animation(.easeInOut(duration: 0.2), value: webSocket.connectionState)
        }

        // Bottom: replay controls (when active)
        if viewModel.showReplayPanel {
            VStack {
                Spacer()
                ReplayControlsView(
                    viewModel: replayViewModel,
                    onStop: {
                        viewModel.toggleReplay()
                    }
                )
                .padding(.horizontal, 12)
                .padding(.bottom, 12)
            }
            .transition(.opacity.combined(with: .move(edge: .bottom)))
            .animation(.easeInOut(duration: 0.2), value: viewModel.showReplayPanel)
        }
    }
}
