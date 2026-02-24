import SwiftUI

/// Filter panel shown below the toolbar.
///
/// Mirrors the web client's `filter-panel.tsx`:
/// - "Show self" toggle
/// - "Show drawings" toggle
/// - Groups section with checkboxes
/// - Users section with checkboxes
/// - "Primary devices only" toggle
struct MapFilterPanel: View {
    @Bindable var viewModel: MapViewModel

    @State private var groupSearch = ""
    @State private var userSearch = ""

    private let searchThreshold = 5

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Master toggles
                VStack(alignment: .leading, spacing: 8) {
                    Toggle("Show self", isOn: $viewModel.showSelf)
                        .font(.subheadline)

                    Toggle("Show drawings", isOn: $viewModel.showDrawings)
                        .font(.subheadline)

                    Toggle("Primary devices only", isOn: $viewModel.primaryOnly)
                        .font(.subheadline)
                }
                .toggleStyle(.switch)

                // Groups section
                if !viewModel.groups.isEmpty {
                    Divider()

                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Groups")
                                .font(.subheadline.weight(.semibold))
                            Spacer()
                            if !viewModel.selectedGroupIds.isEmpty {
                                Button("Clear") {
                                    viewModel.selectedGroupIds.removeAll()
                                }
                                .font(.caption)
                            }
                        }

                        if viewModel.groups.count > searchThreshold {
                            TextField("Search groups...", text: $groupSearch)
                                .textFieldStyle(.roundedBorder)
                                .font(.caption)
                        }

                        ForEach(filteredGroups) { group in
                            groupRow(group)
                        }
                    }
                }

                // Users section
                if !viewModel.users.isEmpty {
                    Divider()

                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Users")
                                .font(.subheadline.weight(.semibold))
                            Spacer()
                            if !viewModel.selectedUserIds.isEmpty {
                                Button("Clear") {
                                    viewModel.selectedUserIds.removeAll()
                                }
                                .font(.caption)
                            }
                        }

                        if viewModel.users.count > searchThreshold {
                            TextField("Search users...", text: $userSearch)
                                .textFieldStyle(.roundedBorder)
                                .font(.caption)
                        }

                        ForEach(filteredUsers) { user in
                            userRow(user)
                        }
                    }
                }
            }
            .padding(12)
        }
        .frame(width: 260, maxHeight: UIScreen.main.bounds.height * 0.6)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
    }

    // MARK: - Rows

    @ViewBuilder
    private func groupRow(_ group: Group) -> some View {
        Button {
            toggleGroup(group.id)
        } label: {
            HStack(spacing: 8) {
                Image(
                    systemName: viewModel.selectedGroupIds.contains(group.id)
                        ? "checkmark.square.fill" : "square")
                    .foregroundStyle(
                        viewModel.selectedGroupIds.contains(group.id) ? .blue : .secondary)
                    .font(.body)

                // Group marker color indicator
                Circle()
                    .fill(Color(hex: group.markerColor) ?? .blue)
                    .frame(width: 10, height: 10)

                Text(group.name)
                    .font(.subheadline)
                    .foregroundStyle(.primary)
                    .lineLimit(1)

                Spacer()
            }
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private func userRow(_ user: User) -> some View {
        Button {
            toggleUser(user.id)
        } label: {
            HStack(spacing: 8) {
                Image(
                    systemName: viewModel.selectedUserIds.contains(user.id)
                        ? "checkmark.square.fill" : "square")
                    .foregroundStyle(
                        viewModel.selectedUserIds.contains(user.id) ? .blue : .secondary)
                    .font(.body)

                Text(user.displayName.isEmpty ? user.username : user.displayName)
                    .font(.subheadline)
                    .foregroundStyle(.primary)
                    .lineLimit(1)

                Spacer()
            }
        }
        .buttonStyle(.plain)
    }

    // MARK: - Computed

    private var filteredGroups: [Group] {
        if groupSearch.isEmpty { return viewModel.groups }
        return viewModel.groups.filter {
            $0.name.localizedCaseInsensitiveContains(groupSearch)
        }
    }

    private var filteredUsers: [User] {
        if userSearch.isEmpty { return viewModel.users }
        return viewModel.users.filter {
            $0.username.localizedCaseInsensitiveContains(userSearch)
                || $0.displayName.localizedCaseInsensitiveContains(userSearch)
        }
    }

    // MARK: - Actions

    private func toggleGroup(_ id: String) {
        if viewModel.selectedGroupIds.contains(id) {
            viewModel.selectedGroupIds.remove(id)
        } else {
            viewModel.selectedGroupIds.insert(id)
        }
    }

    private func toggleUser(_ id: String) {
        if viewModel.selectedUserIds.contains(id) {
            viewModel.selectedUserIds.remove(id)
        } else {
            viewModel.selectedUserIds.insert(id)
        }
    }
}
