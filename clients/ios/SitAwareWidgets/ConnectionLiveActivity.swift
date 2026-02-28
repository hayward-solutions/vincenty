import ActivityKit
import SwiftUI
import WidgetKit

// MARK: - Widget Bundle

@main
struct SitAwareWidgetBundle: WidgetBundle {
    var body: some Widget {
        ConnectionLiveActivityWidget()
    }
}

// MARK: - Live Activity Widget

struct ConnectionLiveActivityWidget: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: ConnectionActivityAttributes.self) { context in
            // Lock Screen / StandBy / banner presentation
            LockScreenLiveActivityView(context: context)
                .activityBackgroundTint(context.state.isConnected ? Color.green.opacity(0.15) : Color.red.opacity(0.15))
                .activitySystemActionForegroundColor(Color.primary)

        } dynamicIsland: { context in
            DynamicIsland {
                // Expanded view (long-press)
                DynamicIslandExpandedRegion(.leading) {
                    HStack(spacing: 6) {
                        ConnectionDot(isConnected: context.state.isConnected, size: 10)
                        Text(context.state.isConnected ? "Connected" : "Reconnecting")
                            .font(.caption.weight(.medium))
                            .foregroundStyle(context.state.isConnected ? Color.green : Color.orange)
                    }
                }
                DynamicIslandExpandedRegion(.trailing) {
                    if let since = context.state.connectedSince, context.state.isConnected {
                        Text(since, style: .timer)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .monospacedDigit()
                    }
                }
                DynamicIslandExpandedRegion(.bottom) {
                    HStack {
                        Image(systemName: "antenna.radiowaves.left.and.right")
                            .foregroundStyle(context.state.isConnected ? .green : .orange)
                        Text(serverHost(from: context.attributes.serverURL))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                        Spacer()
                    }
                    .padding(.horizontal, 4)
                }
            } compactLeading: {
                ConnectionDot(isConnected: context.state.isConnected, size: 8)
            } compactTrailing: {
                Text(context.state.isConnected ? "Live" : "…")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(context.state.isConnected ? .green : .orange)
            } minimal: {
                ConnectionDot(isConnected: context.state.isConnected, size: 8)
            }
            .keylineTint(context.state.isConnected ? .green : .orange)
        }
    }

    /// Extracts just the hostname from a full URL for compact display.
    private func serverHost(from url: String) -> String {
        URL(string: url)?.host ?? url
    }
}

// MARK: - Lock Screen View

private struct LockScreenLiveActivityView: View {
    let context: ActivityViewContext<ConnectionActivityAttributes>

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "antenna.radiowaves.left.and.right")
                .font(.title3)
                .foregroundStyle(context.state.isConnected ? .green : .orange)

            VStack(alignment: .leading, spacing: 2) {
                Text(context.state.isConnected ? "Connected to SitAware" : "Reconnecting…")
                    .font(.subheadline.weight(.semibold))

                if let host = URL(string: context.attributes.serverURL)?.host {
                    Text(host)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer()

            if let since = context.state.connectedSince, context.state.isConnected {
                VStack(alignment: .trailing, spacing: 2) {
                    Text("Connected for")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    Text(since, style: .timer)
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(.primary)
                }
            } else {
                ConnectionDot(isConnected: false, size: 12)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}

// MARK: - Shared Components

private struct ConnectionDot: View {
    let isConnected: Bool
    let size: CGFloat

    var body: some View {
        Circle()
            .fill(isConnected ? Color.green : Color.orange)
            .frame(width: size, height: size)
    }
}
