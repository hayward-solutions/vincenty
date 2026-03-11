import SwiftUI
import LiveKit

struct FeedListView: View {
    let groupId: String
    @Environment(FeedViewModel.self) private var feedVM
    @State private var showAddFeed = false
    @State private var feedName = ""
    @State private var feedType = "rtmp"
    @State private var sourceUrl = ""

    var body: some View {
        Group {
            if feedVM.isViewing, let room = feedVM.room, let feed = feedVM.activeFeed {
                ActiveFeedView(room: room, feed: feed)
            } else {
                feedListContent
            }
        }
        .navigationTitle("Video Feeds")
        .task {
            await feedVM.fetchFeeds(groupId)
        }
    }

    private var feedListContent: some View {
        List {
            Section("Feeds") {
                if feedVM.feeds.isEmpty {
                    Text("No video feeds")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(feedVM.feeds) { feed in
                        HStack {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(feed.name)
                                    .font(.headline)
                                HStack(spacing: 6) {
                                    Text(feed.feedType.uppercased())
                                        .font(.caption2.bold())
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 2)
                                        .background(.blue.opacity(0.2))
                                        .clipShape(Capsule())
                                    Text(feed.isActive ? "Active" : "Inactive")
                                        .font(.caption2)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 2)
                                        .background(feed.isActive ? .green.opacity(0.2) : .gray.opacity(0.2))
                                        .clipShape(Capsule())
                                }
                            }
                            Spacer()
                            if feed.isActive {
                                Button("Watch") {
                                    Task { await feedVM.viewFeed(feed.id) }
                                }
                                .buttonStyle(.borderedProminent)
                            } else {
                                Button("Start") {
                                    Task { _ = await feedVM.startFeed(feed.id) }
                                }
                                .buttonStyle(.bordered)
                            }
                        }
                    }
                }
            }
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    showAddFeed = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showAddFeed) {
            NavigationStack {
                Form {
                    TextField("Feed Name", text: $feedName)
                    Picker("Type", selection: $feedType) {
                        Text("RTMP").tag("rtmp")
                        Text("WHIP").tag("whip")
                        Text("RTSP").tag("rtsp")
                        Text("Phone Camera").tag("phone_cam")
                    }
                    if feedType == "rtsp" {
                        TextField("Source URL", text: $sourceUrl)
                            .textContentType(.URL)
                            .autocapitalization(.none)
                    }
                }
                .navigationTitle("Add Video Feed")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showAddFeed = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Create") {
                            Task {
                                _ = await feedVM.createFeed(
                                    name: feedName,
                                    feedType: feedType,
                                    sourceUrl: feedType == "rtsp" && !sourceUrl.isEmpty ? sourceUrl : nil,
                                    groupId: groupId
                                )
                                showAddFeed = false
                                feedName = ""
                                sourceUrl = ""
                            }
                        }
                        .disabled(feedName.isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }
}

private struct ActiveFeedView: View {
    @Environment(FeedViewModel.self) private var feedVM
    let room: Room
    let feed: VideoFeed

    var body: some View {
        VStack {
            Text(feed.name)
                .font(.title2.bold())

            // Render first remote participant's video track
            if let participant = room.remoteParticipants.values.first,
               let track = participant.firstCameraVideoTrack {
                VideoView(track)
                    .aspectRatio(16/9, contentMode: .fit)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .padding()
            } else {
                VStack(spacing: 12) {
                    Image(systemName: "video.slash")
                        .font(.largeTitle)
                        .foregroundStyle(.secondary)
                    Text("Waiting for feed...")
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, minHeight: 200)
            }

            Spacer()

            Button("Stop Watching") {
                Task { await feedVM.stopViewing() }
            }
            .buttonStyle(.bordered)
            .tint(.red)
        }
        .padding()
    }
}
