import SwiftUI

/// Time range presets matching the web client's `ReplayPanel`.
private enum TimePreset: String, CaseIterable, Identifiable {
    case oneHour        = "1h"
    case sixHours       = "6h"
    case twentyFourHours = "24h"
    case custom         = "Custom"

    var id: String { rawValue }

    var label: String { rawValue }

    /// Resolve the (from, to) date pair for this preset.
    func resolve(customFrom: Date, customTo: Date) -> (from: Date, to: Date)? {
        let now = Date()
        switch self {
        case .oneHour:         return (Calendar.current.date(byAdding: .hour, value: -1,  to: now) ?? now, now)
        case .sixHours:        return (Calendar.current.date(byAdding: .hour, value: -6,  to: now) ?? now, now)
        case .twentyFourHours: return (Calendar.current.date(byAdding: .hour, value: -24, to: now) ?? now, now)
        case .custom:
            guard customTo > customFrom else { return nil }
            let range = customTo.timeIntervalSince(customFrom)
            guard range <= 24 * 3600 else { return nil }
            return (customFrom, customTo)
        }
    }
}

/// Slide-down panel for configuring the replay time range and starting replay.
///
/// Mirrors the web client's `ReplayPanel` (`replay-panel.tsx`).
/// Shown when the replay toolbar button is tapped but no session is active yet.
struct ReplaySetupView: View {
    @Bindable var viewModel: ReplayViewModel
    let scope: ReplayScope
    let onCancel: () -> Void

    @State private var selectedPreset: TimePreset = .oneHour
    @State private var customFrom: Date = Calendar.current.date(byAdding: .hour, value: -1, to: Date()) ?? Date()
    @State private var customTo: Date = Date()
    @State private var validationError: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                Label("Replay", systemImage: "clock.arrow.circlepath")
                    .font(.headline)
                Spacer()
                Button("Cancel", action: onCancel)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            // Preset buttons
            HStack(spacing: 8) {
                ForEach(TimePreset.allCases) { preset in
                    Button {
                        selectedPreset = preset
                        validationError = nil
                    } label: {
                        Text(preset.label)
                            .font(.subheadline.weight(.medium))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(
                                selectedPreset == preset
                                    ? Color.accentColor
                                    : Color.secondary.opacity(0.15)
                            )
                            .foregroundStyle(
                                selectedPreset == preset ? .white : .primary
                            )
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                    .buttonStyle(.plain)
                }
            }

            // Custom date pickers — visible only when Custom is selected
            if selectedPreset == .custom {
                VStack(spacing: 8) {
                    DatePicker("From", selection: $customFrom, displayedComponents: [.date, .hourAndMinute])
                        .datePickerStyle(.compact)
                        .onChange(of: customFrom) { _, _ in validationError = nil }

                    DatePicker("To", selection: $customTo, displayedComponents: [.date, .hourAndMinute])
                        .datePickerStyle(.compact)
                        .onChange(of: customTo) { _, _ in validationError = nil }
                }
                .padding(.vertical, 4)
                .transition(.opacity.combined(with: .move(edge: .top)))
            }

            // Validation error
            if let error = validationError {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            // Fetch error from viewModel
            if let fetchError = viewModel.errorMessage {
                Text(fetchError)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .lineLimit(2)
            }

            // Start button
            Button {
                handleStart()
            } label: {
                HStack(spacing: 6) {
                    if viewModel.isLoading {
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                            .scaleEffect(0.8)
                    } else {
                        Image(systemName: "play.fill")
                    }
                    Text(viewModel.isLoading ? "Loading…" : "Start Replay")
                        .font(.subheadline.weight(.semibold))
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 10)
                .background(Color.accentColor)
                .foregroundStyle(.white)
                .clipShape(RoundedRectangle(cornerRadius: 10))
            }
            .buttonStyle(.plain)
            .disabled(viewModel.isLoading)
        }
        .padding(14)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 14))
        .shadow(color: .black.opacity(0.12), radius: 6, x: 0, y: 3)
        .animation(.easeInOut(duration: 0.2), value: selectedPreset)
    }

    // MARK: - Action

    private func handleStart() {
        guard let range = selectedPreset.resolve(customFrom: customFrom, customTo: customTo) else {
            if selectedPreset == .custom {
                if customTo <= customFrom {
                    validationError = "End time must be after start time."
                } else {
                    validationError = "Custom range cannot exceed 24 hours."
                }
            }
            return
        }

        viewModel.startDate = range.from
        viewModel.endDate   = range.to

        Task {
            await viewModel.start(scope: scope)
        }
    }
}
