import SwiftUI

/// Activity log — shows the current user's own audit log entries.
///
/// Mirrors the web client's `settings/account/activity/page.tsx`:
/// - Audit log table with pagination
/// - Filter by action and resource type
/// - Date range filtering
/// - Export to CSV/JSON
struct ActivityView: View {
    @State private var logs: [AuditLogResponse] = []
    @State private var totalCount = 0
    @State private var currentPage = 1
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Filters
    @State private var selectedAction: String?
    @State private var selectedResourceType: String?
    @State private var showFilters = false

    private let pageSize = 20
    private let api = APIClient.shared

    var body: some View {
        List {
            // Filter toggle
            Section {
                Button {
                    showFilters.toggle()
                } label: {
                    Label(
                        showFilters ? "Hide Filters" : "Show Filters",
                        systemImage: "line.3.horizontal.decrease.circle")
                }
            }

            // Filters
            if showFilters {
                Section("Filters") {
                    Picker("Action", selection: $selectedAction) {
                        Text("All Actions").tag(nil as String?)
                        ForEach(AuditFilters.knownActions, id: \.self) { action in
                            Text(action).tag(action as String?)
                        }
                    }

                    Picker("Resource Type", selection: $selectedResourceType) {
                        Text("All Types").tag(nil as String?)
                        ForEach(AuditFilters.knownResourceTypes, id: \.self) { type in
                            Text(type).tag(type as String?)
                        }
                    }

                    Button("Apply Filters") {
                        currentPage = 1
                        Task { await loadLogs() }
                    }
                    .font(.subheadline)
                }
            }

            // Log entries
            if isLoading {
                Section {
                    ForEach(0..<5, id: \.self) { _ in
                        HStack {
                            RoundedRectangle(cornerRadius: 4)
                                .fill(Color(.systemGray5))
                                .frame(height: 40)
                        }
                    }
                }
            } else if logs.isEmpty {
                Section {
                    Text("No activity found")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Activity Log") {
                    ForEach(logs) { log in
                        logRow(log)
                    }
                }

                // Pagination
                if totalCount > pageSize {
                    Section {
                        HStack {
                            Button("Previous") {
                                currentPage = max(1, currentPage - 1)
                                Task { await loadLogs() }
                            }
                            .disabled(currentPage <= 1)

                            Spacer()

                            let startItem = (currentPage - 1) * pageSize + 1
                            let endItem = min(currentPage * pageSize, totalCount)
                            Text("\(startItem)-\(endItem) of \(totalCount)")
                                .font(.caption)
                                .foregroundStyle(.secondary)

                            Spacer()

                            Button("Next") {
                                currentPage += 1
                                Task { await loadLogs() }
                            }
                            .disabled(currentPage * pageSize >= totalCount)
                        }
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
        .navigationTitle("Activity")
        .task { await loadLogs() }
        .refreshable { await loadLogs() }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Menu {
                    Button {
                        Task { await exportLogs(format: "csv") }
                    } label: {
                        Label("Export CSV", systemImage: "tablecells")
                    }
                    Button {
                        Task { await exportLogs(format: "json") }
                    } label: {
                        Label("Export JSON", systemImage: "curlybraces")
                    }
                } label: {
                    Image(systemName: "square.and.arrow.up")
                }
            }
        }
    }

    // MARK: - Log Row

    @ViewBuilder
    private func logRow(_ log: AuditLogResponse) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 6) {
                Image(systemName: iconForAction(log.action))
                    .font(.caption)
                    .foregroundStyle(.secondary)

                Text(log.action)
                    .font(.subheadline.weight(.medium))

                Spacer()

                Text(formatDate(log.createdAt))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                if !log.resourceType.isEmpty {
                    Text(log.resourceType)
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }

                Text(log.ipAddress)
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 2)
    }

    // MARK: - API Actions

    private func loadLogs() async {
        isLoading = true
        errorMessage = nil

        var filters = AuditFilters()
        filters.action = selectedAction
        filters.resourceType = selectedResourceType
        filters.page = currentPage
        filters.pageSize = pageSize

        do {
            let response: ListResponse<AuditLogResponse> = try await api.get(
                Endpoints.auditLogsMe,
                params: filters.queryParams)
            logs = response.data
            totalCount = response.total
        } catch {
            errorMessage = "Failed to load activity"
        }

        isLoading = false
    }

    private func exportLogs(format: String) async {
        var params: [String: String] = ["format": format]
        if let action = selectedAction { params["action"] = action }
        if let resourceType = selectedResourceType { params["resource_type"] = resourceType }

        do {
            let data = try await api.download(Endpoints.auditLogsMeExport, params: params)

            // Share the exported file
            let ext = format == "csv" ? "csv" : "json"
            let tempURL = FileManager.default.temporaryDirectory
                .appendingPathComponent("activity-export.\(ext)")
            try data.write(to: tempURL)

            // Present share sheet
            await MainActor.run {
                guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
                      let rootVC = windowScene.windows.first?.rootViewController
                else { return }
                let activityVC = UIActivityViewController(
                    activityItems: [tempURL], applicationActivities: nil)
                rootVC.present(activityVC, animated: true)
            }
        } catch {
            errorMessage = "Export failed: \(error.localizedDescription)"
        }
    }

    // MARK: - Helpers

    private func iconForAction(_ action: String) -> String {
        if action.hasPrefix("auth.") { return "person.badge.key" }
        if action.hasPrefix("user.") { return "person" }
        if action.hasPrefix("group.") { return "person.3" }
        if action.hasPrefix("message.") { return "bubble.left" }
        if action.hasPrefix("map_config.") { return "map" }
        return "doc.text"
    }

    private func formatDate(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return iso }
        return date.formatted(date: .abbreviated, time: .shortened)
    }
}
