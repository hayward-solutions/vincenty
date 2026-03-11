import SwiftUI

/// Bottom replay control bar with play/pause, time slider, speed, and time display.
///
/// Mirrors the web client's `replay-controls.tsx`.
struct ReplayControlsView: View {
    @Bindable var viewModel: ReplayViewModel
    let onStop: () -> Void

    @State private var sliderProgress: Double = 0
    @State private var isDragging = false

    var body: some View {
        VStack(spacing: 8) {
            // Time slider
            Slider(
                value: isDragging ? $sliderProgress : .constant(viewModel.progress),
                in: 0...1,
                onEditingChanged: { editing in
                    isDragging = editing
                    if !editing {
                        viewModel.seek(to: sliderProgress)
                    }
                }
            )
            .tint(.blue)

            // Controls row
            HStack(spacing: 16) {
                // Stop button
                Button {
                    onStop()
                } label: {
                    Image(systemName: "stop.fill")
                        .font(.system(size: 14))
                }

                // Play/Pause
                Button {
                    if viewModel.isPlaying {
                        viewModel.pause()
                    } else {
                        viewModel.play()
                    }
                } label: {
                    Image(systemName: viewModel.isPlaying ? "pause.fill" : "play.fill")
                        .font(.system(size: 18))
                }

                // Current time display
                Text(formatDate(viewModel.currentTime))
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.secondary)

                Spacer()

                // Speed picker
                Menu {
                    ForEach(ReplaySpeed.allCases, id: \.rawValue) { speed in
                        Button {
                            viewModel.speed = speed
                        } label: {
                            HStack {
                                Text(speed.label)
                                if viewModel.speed == speed {
                                    Image(systemName: "checkmark")
                                }
                            }
                        }
                    }
                } label: {
                    Text(viewModel.speed.label)
                        .font(.caption.weight(.medium))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(.secondary.opacity(0.15))
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                }

                // Entry count
                Text("\(viewModel.visibleEntries.count)/\(viewModel.historyEntries.count)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.horizontal)
        .padding(.vertical, 10)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: -2)
        .onChange(of: viewModel.progress) { _, newValue in
            if !isDragging {
                sliderProgress = newValue
            }
        }
    }

    private func formatDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"
        return formatter.string(from: date)
    }
}
