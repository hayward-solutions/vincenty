import SwiftUI
import CoreLocation

/// The main map screen — full-viewport MapLibre map with overlaid controls.
///
/// Mirrors the web client's `app/(app)/map/page.tsx` layout:
/// - MapContainerView as the base layer
/// - MapToolbarView top-left
/// - MapControlsView top-right
/// - MapFilterPanel below toolbar (when active)
/// - ReplaySetupView below toolbar (when replay is open but not yet active)
/// - MeasurePanelView below toolbar (when measure active)
/// - DrawPanelView below toolbar (when draw active)
/// - ReplayControlsView at bottom (when replay is active / playing)
/// - Loading spinner while settings load
///
/// The `.onChange` modifier chain is split across three computed properties
/// (`coreView` → `mapToolView` → `body`) to prevent the Swift type-checker
/// from timing out on a single large expression.
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
    @State private var historyTracksController = HistoryTracksController()

    // MARK: - Body

    var body: some View {
        mapToolView
            // Phase 1: replay session activated — data ready, create stable layers.
            .onChange(of: replayViewModel.isActive) { _, isNowActive in
                if isNowActive {
                    historyTracksController.setupLayers(
                        allEntries: replayViewModel.historyEntries)
                    historyTracksController.updateData(
                        visibleEntries: replayViewModel.visibleEntries)
                }
            }
            // Phase 2: playback cursor moved — update source data only (no layer churn).
            .onChange(of: replayViewModel.currentTime) { _, _ in
                guard replayViewModel.isActive else { return }
                historyTracksController.updateData(
                    visibleEntries: replayViewModel.visibleEntries)
            }
    }

    // MARK: - Map Tool View (draw + measure onChange)

    /// Applies draw/measure tool lifecycle modifiers on top of `coreView`.
    private var mapToolView: some View {
        coreView
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
            .onChange(of: viewModel.showDrawPanel) { _, isActive in
                if isActive {
                    drawController.activate(
                        mode: drawingsViewModel.drawMode,
                        style: drawingsViewModel.drawStyle)
                    drawController.onShapeComplete = { feature, mode in
                        drawingsViewModel.completedShapes.append(
                            CompletedShape(feature: feature, shapeType: mode))
                    }
                    drawController.updateCompletedShapes(drawingsViewModel.completedShapes)
                } else {
                    drawController.deactivate()
                }
            }
            .onChange(of: drawingsViewModel.completedShapes.count) { _, _ in
                if drawController.isActive {
                    drawController.updateCompletedShapes(drawingsViewModel.completedShapes)
                }
            }
            .onChange(of: drawingsViewModel.drawMode) { _, newMode in
                if drawController.isActive { drawController.updateMode(newMode) }
            }
            .onChange(of: drawingsViewModel.drawStyle) { _, newStyle in
                if drawController.isActive { drawController.updateStyle(newStyle) }
            }
    }

    // MARK: - Core View (ZStack + tasks + location/replay onChange)

    /// ZStack containing the map and overlays, plus task and location modifiers.
    private var coreView: some View {
        ZStack {
            if viewModel.isLoadingSettings {
                LoadingStateView(message: "Loading map...", style: .fullScreen)
            } else {
                mapContainerView
                mapOverlays
            }
        }
        .task { await viewModel.loadInitialData() }
        .task { viewModel.subscribeToLocations(webSocket: webSocket) }
        .task {
            if let deviceId = deviceManager.deviceId {
                locationSharing.startSharing(webSocket: webSocket, deviceId: deviceId)
            }
        }
        .onDisappear {
            viewModel.unsubscribe()
            measureController.deactivate()
            drawController.deactivate()
        }
        // Live location markers — suppressed while replay is active.
        .onChange(of: viewModel.displayLocations) { _, locations in
            guard !replayViewModel.isActive else { return }
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
        // Replay panel toggle: tear down session and restore live markers on close.
        .onChange(of: viewModel.showReplayPanel) { _, isOpen in
            if !isOpen {
                replayViewModel.stop()
                historyTracksController.removeAll()
                locationMarkers.update(
                    locations: viewModel.displayLocations,
                    currentDeviceId: deviceManager.deviceId,
                    groups: viewModel.groups)
            }
        }
    }

    // MARK: - Location Updates

    private func updateSelfPosition() {
        viewModel.selfPosition = locationSharing.currentPosition
        selfMarker.update(
            position: viewModel.showSelf ? viewModel.selfPosition : nil,
            autoCenter: !replayViewModel.isActive)
        viewModel.updateTrackingIfNeeded()
    }

    // MARK: - Tap Handling

    private func handleMapTap(_ coordinate: CLLocationCoordinate2D) {
        if measureController.isActive {
            measureController.handleTap(at: coordinate)
        } else if drawController.isActive {
            drawController.handleTap(at: coordinate)
        }
    }

    private func handleMapDoubleTap(_ coordinate: CLLocationCoordinate2D) {
        if measureController.isActive {
            measureController.handleDoubleTap(at: coordinate)
        } else if drawController.isActive {
            drawController.handleDoubleTap(at: coordinate)
        }
    }

    // MARK: - Replay Scope

    /// Auto-detect scope from current filter selections, mirroring the web client's logic.
    private var resolvedReplayScope: ReplayScope {
        if viewModel.selectedUserIds.count == 1,
           let userId = viewModel.selectedUserIds.first {
            return .user(userId)
        } else if viewModel.selectedGroupIds.count == 1,
                  let groupId = viewModel.selectedGroupIds.first {
            return .group(groupId)
        }
        return .all
    }

    // MARK: - Map Container

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
                historyTracksController.attach(to: mapView)
                updateSelfPosition()
                if !replayViewModel.isActive {
                    locationMarkers.update(
                        locations: viewModel.displayLocations,
                        currentDeviceId: deviceManager.deviceId,
                        groups: viewModel.groups)
                }
            },
            onCameraChanged: { bearing, pitch, zoom in
                viewModel.onCameraChanged(bearing: bearing, pitch: pitch, zoom: zoom)
            },
            onUserDragBegan: { viewModel.onUserDragBegan() },
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

            // Replay setup: shown when the panel is open but no session is active yet.
            if viewModel.showReplayPanel && !replayViewModel.isActive {
                ReplaySetupView(
                    viewModel: replayViewModel,
                    scope: resolvedReplayScope,
                    onCancel: { viewModel.showReplayPanel = false }
                )
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
                    onClose: { viewModel.toggleMeasure() }
                )
                .transition(.opacity.combined(with: .move(edge: .top)))
            }

            if viewModel.showDrawPanel {
                DrawPanelView(
                    viewModel: drawingsViewModel,
                    groups: viewModel.groups,
                    onClose: { viewModel.toggleDraw() }
                )
                .transition(.opacity.combined(with: .move(edge: .top)))
            }

            Spacer()
        }
        .padding(.leading, 12)
        .padding(.top, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .animation(.easeInOut(duration: 0.2), value: viewModel.showFilterPanel)
        .animation(.easeInOut(duration: 0.2), value: viewModel.showReplayPanel)
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

        // Top-center: WebSocket status banner (hidden when connected)
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

        // Bottom: replay playback controls (shown once the session is active)
        if replayViewModel.isActive {
            VStack {
                Spacer()
                ReplayControlsView(
                    viewModel: replayViewModel,
                    onStop: { viewModel.showReplayPanel = false }
                )
                .padding(.horizontal, 12)
                .padding(.bottom, 12)
            }
            .transition(.opacity.combined(with: .move(edge: .bottom)))
            .animation(.easeInOut(duration: 0.2), value: replayViewModel.isActive)
        }
    }
}
