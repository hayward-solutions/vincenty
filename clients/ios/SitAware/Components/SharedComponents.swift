import SwiftUI

// MARK: - Error Banner

/// Dismissible error banner displayed within a `List` or `Form` section.
///
/// Extracts the repeated `Section { Text(error).foregroundStyle(.red).font(.caption) }` pattern
/// used across `AdminGroupsView`, `DevicesView`, `ActivityView`, etc.
struct ErrorBanner: View {
    let message: String
    var onDismiss: (() -> Void)?

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(.red)
                .font(.caption)

            Text(message)
                .font(.caption)
                .foregroundStyle(.red)
                .frame(maxWidth: .infinity, alignment: .leading)

            if let onDismiss {
                Button {
                    onDismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Dismiss error")
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Error: \(message)")
    }
}

// MARK: - Loading State View

/// Centered loading indicator for full-screen or section loading states.
///
/// Replaces the `ProgressView().frame(maxWidth: .infinity)` and `VStack { ProgressView(); Text(...) }`
/// patterns used in `ContentView`, `MapScreen`, list views, etc.
struct LoadingStateView: View {
    var message: String?
    var style: LoadingStyle = .inline

    enum LoadingStyle {
        /// Compact spinner for use in `List` sections.
        case inline
        /// Full-screen centered spinner with optional message.
        case fullScreen
    }

    var body: some View {
        switch style {
        case .inline:
            ProgressView()
                .frame(maxWidth: .infinity)
                .accessibilityLabel(message ?? "Loading")

        case .fullScreen:
            VStack(spacing: 12) {
                ProgressView()
                if let message {
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .accessibilityElement(children: .combine)
            .accessibilityLabel(message ?? "Loading")
        }
    }
}

// MARK: - Empty State View

/// Placeholder view for empty lists/collections.
///
/// Replaces the `Text("No items found").foregroundStyle(.secondary)` pattern
/// used across `AdminGroupsView`, `DevicesView`, `ActivityView`, etc.
/// Also provides an optional action button (e.g., "Create Group").
struct EmptyStateView: View {
    let title: String
    var systemImage: String = "tray"
    var description: String?
    var actionTitle: String?
    var action: (() -> Void)?

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 32))
                .foregroundStyle(.secondary)

            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.secondary)

            if let description {
                Text(description)
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .multilineTextAlignment(.center)
            }

            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .font(.subheadline)
                    .buttonStyle(.bordered)
                    .padding(.top, 4)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 20)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title). \(description ?? "")")
    }
}

// MARK: - Badge View

/// Small colored pill badge used for status labels, permission badges, and role indicators.
///
/// Extracts the repeated pattern from `AdminGroupDetailView.permBadge`,
/// `DevicesView` (This device / Primary), `AdminUsersView` (Admin/Active/MFA badges).
struct BadgeView: View {
    let text: String
    var color: Color = .blue
    var style: BadgeStyle = .subtle

    enum BadgeStyle {
        /// Light background with colored text (default).
        case subtle
        /// Filled background with white text.
        case filled
        /// Outline with colored border and text.
        case outline
    }

    var body: some View {
        Text(text)
            .font(.caption2.weight(.medium))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(badgeBackground)
            .foregroundStyle(badgeForeground)
            .clipShape(RoundedRectangle(cornerRadius: 4))
            .accessibilityLabel(text)
    }

    @ViewBuilder
    private var badgeBackground: some View {
        switch style {
        case .subtle:
            color.opacity(0.15)
        case .filled:
            color
        case .outline:
            RoundedRectangle(cornerRadius: 4)
                .stroke(color, lineWidth: 1)
                .background(Color.clear)
        }
    }

    private var badgeForeground: Color {
        switch style {
        case .subtle, .outline:
            color
        case .filled:
            .white
        }
    }
}

// MARK: - Async Button

/// Button that executes an async action and shows a loading indicator while running.
///
/// Extracts the repeated pattern from form sheets and action buttons:
/// ```
/// Button { Task { await doSomething() } } label: {
///     if isLoading { ProgressView() } else { Text("Submit") }
/// }.disabled(isLoading || !isValid)
/// ```
struct AsyncButton<Label: View>: View {
    let action: () async -> Void
    @ViewBuilder let label: () -> Label

    @State private var isRunning = false

    var body: some View {
        Button {
            guard !isRunning else { return }
            isRunning = true
            Task {
                await action()
                isRunning = false
            }
        } label: {
            if isRunning {
                ProgressView()
                    .frame(maxWidth: .infinity)
            } else {
                label()
            }
        }
        .disabled(isRunning)
    }
}

// MARK: - Pagination Controls

/// Reusable pagination controls (Previous / "1-20 of 100" / Next).
///
/// Extracts the repeated pagination `HStack` from `AdminGroupsView`,
/// `AdminUsersView`, `AdminAuditLogsView`, `ActivityView`.
struct PaginationControls: View {
    let currentPage: Int
    let pageSize: Int
    let totalCount: Int
    let onPageChange: (Int) -> Void

    private var totalPages: Int {
        max(1, Int(ceil(Double(totalCount) / Double(pageSize))))
    }

    private var startItem: Int {
        (currentPage - 1) * pageSize + 1
    }

    private var endItem: Int {
        min(currentPage * pageSize, totalCount)
    }

    var body: some View {
        HStack {
            Button("Previous") {
                onPageChange(max(1, currentPage - 1))
            }
            .disabled(currentPage <= 1)

            Spacer()

            Text("\(startItem)-\(endItem) of \(totalCount)")
                .font(.caption)
                .foregroundStyle(.secondary)

            Spacer()

            Button("Next") {
                onPageChange(currentPage + 1)
            }
            .disabled(currentPage >= totalPages)
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Page \(currentPage) of \(totalPages)")
    }
}

// MARK: - Skeleton Row

/// Placeholder shimmer row for loading states.
///
/// Used by `ActivityView` and `AdminAuditLogsView` as skeleton loading indicators.
struct SkeletonRow: View {
    var height: CGFloat = 40

    var body: some View {
        RoundedRectangle(cornerRadius: 4)
            .fill(.gray.opacity(0.15))
            .frame(height: height)
            .accessibilityHidden(true)
    }
}

// MARK: - Status Banner

/// Floating capsule-style banner for connectivity/sync status indicators.
///
/// Extracts the offline banner (red) and WS connecting banner (orange)
/// patterns from `ContentView`.
struct StatusBanner: View {
    let icon: String?
    let message: String
    var color: Color = .red
    var showSpinner: Bool = false
    var edge: VerticalAlignment = .top

    var body: some View {
        HStack(spacing: 6) {
            if showSpinner {
                ProgressView()
                    .controlSize(.mini)
            }
            if let icon {
                Image(systemName: icon)
                    .font(.caption2)
            }
            Text(message)
                .font(.caption2.weight(.medium))
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(color.opacity(0.9))
        .foregroundStyle(.white)
        .clipShape(Capsule())
        .accessibilityElement(children: .combine)
        .accessibilityLabel(message)
        .accessibilityAddTraits(.updatesFrequently)
    }
}

// MARK: - Confirmation Alert Helper

/// A destructive confirmation alert modifier that consolidates the repeated
/// `.alert("Title", isPresented:) { Button("Cancel"); Button("Action", role: .destructive) }` pattern.
struct DestructiveConfirmation: ViewModifier {
    let title: String
    @Binding var isPresented: Bool
    let message: String
    let actionTitle: String
    let action: () -> Void

    func body(content: Content) -> some View {
        content.alert(title, isPresented: $isPresented) {
            Button("Cancel", role: .cancel) {}
            Button(actionTitle, role: .destructive, action: action)
        } message: {
            Text(message)
        }
    }
}

extension View {
    /// Attach a destructive confirmation alert to any view.
    ///
    /// ```swift
    /// .destructiveConfirmation(
    ///     "Delete Group",
    ///     isPresented: $showDeleteAlert,
    ///     message: "This will remove all members.",
    ///     actionTitle: "Delete") { deleteGroup() }
    /// ```
    func destructiveConfirmation(
        _ title: String,
        isPresented: Binding<Bool>,
        message: String,
        actionTitle: String = "Delete",
        action: @escaping () -> Void
    ) -> some View {
        modifier(DestructiveConfirmation(
            title: title,
            isPresented: isPresented,
            message: message,
            actionTitle: actionTitle,
            action: action))
    }
}
