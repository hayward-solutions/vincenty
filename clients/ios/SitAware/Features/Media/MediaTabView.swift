import SwiftUI

struct MediaScreen: View {
    @State private var selectedSection = MediaSection.live

    enum MediaSection: String, CaseIterable {
        case live = "Live"
        case feeds = "Feeds"
        case ptt = "PTT"
        case calls = "Calls"
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Section picker
                Picker("Section", selection: $selectedSection) {
                    ForEach(MediaSection.allCases, id: \.self) { section in
                        Text(section.rawValue).tag(section)
                    }
                }
                .pickerStyle(.segmented)
                .padding()

                // Content
                switch selectedSection {
                case .live:
                    LiveMonitorView()
                case .feeds:
                    GroupFeedPickerView()
                case .ptt:
                    GroupPTTPickerView()
                case .calls:
                    CallScreen()
                }
            }
            .navigationTitle("Media")
        }
    }
}

/// Lets the user pick a group, then shows PTT channels for that group.
private struct GroupPTTPickerView: View {
    @State private var groups: [Group] = []
    @State private var selectedGroup: Group?
    @State private var isLoading = true

    var body: some View {
        if let group = selectedGroup {
            PTTView(groupId: group.id)
                .toolbar {
                    ToolbarItem(placement: .navigation) {
                        Button("Groups") { selectedGroup = nil }
                    }
                }
        } else {
            List {
                if isLoading {
                    ProgressView()
                } else if groups.isEmpty {
                    Text("No groups available")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(groups) { group in
                        Button {
                            selectedGroup = group
                        } label: {
                            Text(group.name)
                        }
                    }
                }
            }
            .task {
                do {
                    groups = try await APIClient.shared.get(Endpoints.usersMeGroups)
                    isLoading = false
                } catch {
                    isLoading = false
                }
            }
        }
    }
}

/// Lets the user pick a group, then shows feeds for that group.
private struct GroupFeedPickerView: View {
    @State private var groups: [Group] = []
    @State private var selectedGroup: Group?
    @State private var isLoading = true

    var body: some View {
        if let group = selectedGroup {
            FeedListView(groupId: group.id)
                .toolbar {
                    ToolbarItem(placement: .navigation) {
                        Button("Groups") { selectedGroup = nil }
                    }
                }
        } else {
            List {
                if isLoading {
                    ProgressView()
                } else if groups.isEmpty {
                    Text("No groups available")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(groups) { group in
                        Button {
                            selectedGroup = group
                        } label: {
                            Text(group.name)
                        }
                    }
                }
            }
            .task {
                do {
                    groups = try await APIClient.shared.get(Endpoints.usersMeGroups)
                    isLoading = false
                } catch {
                    isLoading = false
                }
            }
        }
    }
}
