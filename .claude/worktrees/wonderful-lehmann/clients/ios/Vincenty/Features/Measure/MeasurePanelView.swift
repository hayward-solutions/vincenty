import SwiftUI

/// Measure panel shown below the toolbar when the measure tool is active.
///
/// Mirrors the web client's `measure-panel.tsx`:
/// - Mode toggle (Distance / Radius)
/// - Measurement display (total distance, or radius + area)
/// - Clear button to reset points without deactivating
/// - Close button to deactivate the tool
struct MeasurePanelView: View {
    var mode: MeasureMode
    var measurements: MeasureResult
    var onModeChange: (MeasureMode) -> Void
    var onClear: () -> Void
    var onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                Text("Measure")
                    .font(.headline)
                Spacer()
                Button { onClose() } label: {
                    Image(systemName: "xmark")
                        .font(.subheadline)
                }
                .accessibilityLabel("Close measure panel")
            }

            // Mode selector
            HStack(spacing: 8) {
                modeButton(title: "Distance", icon: "ruler", targetMode: .line)
                modeButton(title: "Radius", icon: "circle.dashed", targetMode: .circle)
            }

            // Measurement display
            if hasMeasurements {
                measurementContent
            } else {
                instructionText
            }

            // Clear button (only if there are measurements)
            if hasMeasurements {
                Button {
                    onClear()
                } label: {
                    HStack {
                        Image(systemName: "trash")
                            .font(.caption)
                        Text("Clear")
                            .font(.caption)
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
            }
        }
        .padding(12)
        .frame(width: 260)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
    }

    // MARK: - Mode Button

    @ViewBuilder
    private func modeButton(title: String, icon: String, targetMode: MeasureMode) -> some View {
        Button {
            onModeChange(targetMode)
        } label: {
            VStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 16))
                Text(title)
                    .font(.caption2)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(
                mode == targetMode
                    ? Color.accentColor.opacity(0.15)
                    : Color.clear)
            .clipShape(RoundedRectangle(cornerRadius: 8))
        }
        .foregroundStyle(mode == targetMode ? .primary : .secondary)
    }

    // MARK: - Content

    private var hasMeasurements: Bool {
        measurements.total > 0
    }

    @ViewBuilder
    private var measurementContent: some View {
        switch mode {
        case .line:
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text("Total Distance")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                    Spacer()
                }
                Text(MeasureToolController.formatDistance(measurements.total))
                    .font(.title2.weight(.semibold).monospacedDigit())

                // Segment breakdown (if multiple segments)
                if measurements.segments.count > 1 {
                    Divider()
                    VStack(alignment: .leading, spacing: 2) {
                        ForEach(Array(measurements.segments.enumerated()), id: \.offset) { idx, segment in
                            HStack {
                                Text("Segment \(idx + 1)")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                                Spacer()
                                Text(MeasureToolController.formatDistance(segment))
                                    .font(.caption.monospacedDigit())
                            }
                        }
                    }
                }
            }

        case .circle:
            VStack(alignment: .leading, spacing: 8) {
                // Radius
                VStack(alignment: .leading, spacing: 2) {
                    Text("Radius")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                    Text(MeasureToolController.formatDistance(measurements.radius ?? 0))
                        .font(.title2.weight(.semibold).monospacedDigit())
                }

                // Area
                if let area = measurements.area {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Area")
                            .font(.caption.weight(.medium))
                            .foregroundStyle(.secondary)
                        Text(MeasureToolController.formatArea(area))
                            .font(.title3.weight(.semibold).monospacedDigit())
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var instructionText: some View {
        Text(mode == .line
             ? "Tap points on the map to measure distance"
             : "Tap center, then tap edge to measure radius")
            .font(.caption)
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .center)
            .padding(.vertical, 8)
    }
}
