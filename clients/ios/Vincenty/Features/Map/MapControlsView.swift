import SwiftUI

/// Vertical control panel on the right side of the map.
///
/// Mirrors the web client's `map-controls.tsx`:
/// - Zoom In / Zoom Out
/// - Reset North (compass arrow, rotates with bearing)
/// - Toggle Terrain (if available)
/// - Track My Location (fly to self + continuous following)
struct MapControlsView: View {
    @Bindable var viewModel: MapViewModel

    var body: some View {
        VStack(spacing: 0) {
            // Zoom In
            controlButton(icon: "plus", label: "Zoom in", action: viewModel.zoomIn)

            Divider()

            // Zoom Out
            controlButton(icon: "minus", label: "Zoom out", action: viewModel.zoomOut)

            Divider()

            // Reset North / Compass
            Button {
                viewModel.resetNorth()
            } label: {
                Image(systemName: "location.north.fill")
                    .font(.system(size: 16, weight: .medium))
                    .rotationEffect(.degrees(-viewModel.bearing))
                    .foregroundStyle(
                        viewModel.bearing != 0 || viewModel.pitch != 0
                            ? AnyShapeStyle(.tint) : AnyShapeStyle(.primary))
                    .frame(width: 44, height: 44)
            }
            .accessibilityLabel("Reset north")
            .accessibilityHint("Double-tap to reset map orientation to north")

            Divider()

            // Terrain toggle (only if terrain source available)
            if viewModel.terrainAvailable {
                Button {
                    viewModel.toggleTerrain()
                } label: {
                    Image(systemName: viewModel.terrainEnabled ? "mountain.2.fill" : "mountain.2")
                        .font(.system(size: 16, weight: .medium))
                        .foregroundStyle(viewModel.terrainEnabled ? AnyShapeStyle(.tint) : AnyShapeStyle(.primary))
                        .frame(width: 44, height: 44)
                }
                .accessibilityLabel("Terrain")
                .accessibilityValue(viewModel.terrainEnabled ? "On" : "Off")
                .accessibilityHint("Double-tap to toggle 3D terrain")

                Divider()
            }

            // Track My Location
            Button {
                viewModel.flyToSelf()
            } label: {
                Image(
                    systemName: viewModel.isTracking
                        ? "location.fill" : "location")
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(viewModel.isTracking ? AnyShapeStyle(.tint) : AnyShapeStyle(.primary))
                    .frame(width: 44, height: 44)
            }
            .accessibilityLabel("My location")
            .accessibilityValue(viewModel.isTracking ? "Tracking" : "Not tracking")
            .accessibilityHint("Double-tap to center map on your location")
        }
        .glassEffect(.regular, in: .rect(cornerRadius: 10))
        .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
        .fixedSize(horizontal: true, vertical: false)
    }

    @ViewBuilder
    private func controlButton(icon: String, label: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 16, weight: .medium))
                .frame(width: 44, height: 44)
        }
        .accessibilityLabel(label)
    }
}
