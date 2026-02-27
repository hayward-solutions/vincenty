import SwiftUI

/// Groups list — shows groups the current user belongs to, with marker editing for group admins.
///
/// Mirrors the web client's `settings/account/groups/page.tsx`:
/// - List of user's groups with marker preview
/// - "Edit Marker" button for group admins
/// - Shape and color picker in a sheet
struct GroupsView: View {
    @Environment(AuthManager.self) private var auth

    @State private var groups: [Group] = []
    @State private var groupAdminStatus: [String: Bool] = [:]  // groupId → isAdmin
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Marker editing
    @State private var editingGroup: Group?
    @State private var editMarkerIcon = "default"
    @State private var editMarkerColor = "#3b82f6"
    @State private var showMarkerEditor = false
    @State private var isSavingMarker = false

    private let api = APIClient.shared

    var body: some View {
        List {
            if isLoading {
                Section {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                }
            } else if groups.isEmpty {
                Section {
                    Text("You are not a member of any groups")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Your Groups") {
                    ForEach(groups) { group in
                        groupRow(group)
                    }
                }
            }

            if let error = errorMessage {
                Section {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
            }
        }
        .navigationTitle("Groups")
        .task { await loadGroups() }
        .refreshable { await loadGroups() }
        .sheet(isPresented: $showMarkerEditor) { markerEditorSheet }
    }

    // MARK: - Group Row

    @ViewBuilder
    private func groupRow(_ group: Group) -> some View {
        HStack(spacing: 12) {
            // Marker preview
            Circle()
                .fill(Color(hex: group.markerColor.isEmpty ? "#3b82f6" : group.markerColor))
                .frame(width: 32, height: 32)
                .overlay(
                    Text(String(group.name.prefix(1)).uppercased())
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.white))

            VStack(alignment: .leading, spacing: 2) {
                Text(group.name)
                    .font(.subheadline.weight(.medium))

                let desc = group.description
                if !desc.isEmpty {
                    Text(desc)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }

                HStack(spacing: 6) {
                    if !group.markerIcon.isEmpty {
                        Text(group.markerIcon.capitalized)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(
                                Color(hex: group.markerColor).opacity(0.15)
                            )
                            .clipShape(RoundedRectangle(cornerRadius: 4))
                    }
                }
            }

            Spacer()

            // Edit marker button (only for group admins)
            if groupAdminStatus[group.id] == true {
                Button {
                    editingGroup = group
                    editMarkerIcon = group.markerIcon.isEmpty ? "default" : group.markerIcon
                    editMarkerColor = group.markerColor.isEmpty ? "#3b82f6" : group.markerColor
                    showMarkerEditor = true
                } label: {
                    Label("Edit Marker", systemImage: "paintbrush")
                        .font(.caption)
                }
            } else {
                Text("Admin only")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Marker Editor Sheet

    @ViewBuilder
    private var markerEditorSheet: some View {
        NavigationStack {
            Form {
                // Live preview
                Section {
                    HStack {
                        Spacer()
                        Circle()
                            .fill(Color(hex: editMarkerColor))
                            .frame(width: 48, height: 48)
                            .overlay(
                                Text(String(editingGroup?.name.prefix(1) ?? "G").uppercased())
                                    .font(.headline.weight(.bold))
                                    .foregroundStyle(.white))
                        Spacer()
                    }
                }

                // Shape picker
                Section("Shape") {
                    LazyVGrid(
                        columns: Array(repeating: GridItem(.flexible()), count: 5),
                        spacing: 8
                    ) {
                        ForEach(ProfileView.availableShapes, id: \.self) { shape in
                            Button {
                                editMarkerIcon = shape
                            } label: {
                                Text(shape.capitalized)
                                    .font(.system(size: 10))
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 8)
                                    .background(
                                        editMarkerIcon == shape
                                            ? Color.accentColor.opacity(0.15)
                                            : Color(.tertiarySystemGroupedBackground))
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                            .foregroundStyle(editMarkerIcon == shape ? .primary : .secondary)
                        }
                    }
                }

                // Color picker
                Section("Color") {
                    HStack(spacing: 6) {
                        ForEach(ProfileView.presetColors, id: \.self) { color in
                            Button {
                                editMarkerColor = color
                            } label: {
                                RoundedRectangle(cornerRadius: 4)
                                    .fill(Color(hex: color))
                                    .frame(width: 26, height: 26)
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 4)
                                            .strokeBorder(
                                                editMarkerColor == color ? Color.primary : .clear,
                                                lineWidth: 2))
                            }
                        }
                    }
                }
            }
            .navigationTitle("Edit Marker")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showMarkerEditor = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await saveGroupMarker() }
                    }
                    .disabled(isSavingMarker)
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadGroups() async {
        isLoading = true
        errorMessage = nil

        do {
            let fetchedGroups: [Group] = try await api.get(Endpoints.usersMeGroups)
            groups = fetchedGroups

            // Check admin status for each group
            await withTaskGroup(of: (String, Bool).self) { taskGroup in
                for group in fetchedGroups {
                    taskGroup.addTask {
                        await (group.id, self.checkGroupAdmin(groupId: group.id))
                    }
                }
                for await (groupId, isAdmin) in taskGroup {
                    groupAdminStatus[groupId] = isAdmin
                }
            }
        } catch {
            errorMessage = "Failed to load groups"
        }

        isLoading = false
    }

    private func checkGroupAdmin(groupId: String) async -> Bool {
        guard let userId = auth.user?.id else { return false }
        do {
            let members: [GroupMember] = try await api.get(
                Endpoints.groupMembers(groupId))
            if let member = members.first(where: { $0.userId == userId }) {
                return member.isGroupAdmin
            }
            return false
        } catch {
            return false
        }
    }

    private func saveGroupMarker() async {
        guard let group = editingGroup else { return }
        isSavingMarker = true

        do {
            struct MarkerBody: Encodable {
                let markerIcon: String
                let markerColor: String
            }
            let _: Group = try await api.put(
                Endpoints.groupMarker(group.id),
                body: MarkerBody(markerIcon: editMarkerIcon, markerColor: editMarkerColor))
            showMarkerEditor = false
            await loadGroups()
        } catch {
            errorMessage = "Failed to save marker"
        }

        isSavingMarker = false
    }
}

