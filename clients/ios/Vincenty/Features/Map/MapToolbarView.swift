import SwiftUI

/// Horizontal toolbar at the top-left of the map.
///
/// Mirrors the web client's `map-toolbar.tsx`:
/// - Replay (history icon)
/// - Filter (filter icon)
/// - Measure (ruler icon)
/// - Draw (pencil icon)
///
/// Mutual exclusion of panels is handled by the view model.
struct MapToolbarView: View {
    @Bindable var viewModel: MapViewModel

    var body: some View {
        HStack(spacing: 0) {
            // Replay
            toolbarButton(
                icon: "clock.arrow.circlepath",
                isActive: viewModel.showReplayPanel,
                label: "Replay",
                action: viewModel.toggleReplay)

            Divider()
                .frame(height: 24)

            // Filters
            toolbarButton(
                icon: "line.3.horizontal.decrease",
                isActive: viewModel.showFilterPanel,
                label: "Filters",
                action: viewModel.toggleFilter)

            Divider()
                .frame(height: 24)

            // Measure
            toolbarButton(
                icon: "ruler",
                isActive: viewModel.showMeasurePanel,
                label: "Measure",
                action: viewModel.toggleMeasure)

            Divider()
                .frame(height: 24)

            // Draw
            toolbarButton(
                icon: "pencil.and.outline",
                isActive: viewModel.showDrawPanel,
                label: "Draw",
                action: viewModel.toggleDraw)
        }
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
    }

    @ViewBuilder
    private func toolbarButton(
        icon: String, isActive: Bool, label: String, action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 16, weight: .medium))
                .foregroundStyle(isActive ? .primary : .secondary)
                .frame(width: 44, height: 44)
        }
        .accessibilityLabel(label)
        .accessibilityValue(isActive ? "Active" : "Inactive")
        .accessibilityHint("Double-tap to \(isActive ? "close" : "open") \(label.lowercased()) panel")
    }
}
