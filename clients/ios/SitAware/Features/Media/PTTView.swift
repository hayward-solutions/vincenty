import SwiftUI

struct PTTView: View {
    let groupId: String
    @Environment(PTTViewModel.self) private var pttVM
    @Environment(WebSocketService.self) private var webSocket
    @State private var showCreateChannel = false
    @State private var channelName = ""
    @State private var isDefault = false

    var body: some View {
        Group {
            if pttVM.isConnected, let channel = pttVM.activeChannel {
                ActivePTTView(channel: channel)
            } else {
                channelListView
            }
        }
        .navigationTitle("Push to Talk")
        .task {
            await pttVM.fetchChannels(groupId)
            pttVM.subscribeToFloorEvents(webSocket: webSocket)
        }
        .onDisappear {
            pttVM.unsubscribe()
        }
    }

    private var channelListView: some View {
        List {
            Section("Channels") {
                if pttVM.channels.isEmpty {
                    Text("No PTT channels")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(pttVM.channels) { channel in
                        HStack {
                            VStack(alignment: .leading) {
                                HStack {
                                    Text(channel.name)
                                        .font(.headline)
                                    if channel.isDefault {
                                        Text("Default")
                                            .font(.caption2)
                                            .padding(.horizontal, 6)
                                            .padding(.vertical, 2)
                                            .background(.blue.opacity(0.2))
                                            .clipShape(Capsule())
                                    }
                                }
                            }
                            Spacer()
                            Button("Join") {
                                Task { await pttVM.joinChannel(groupId, channelId: channel.id) }
                            }
                            .buttonStyle(.borderedProminent)
                        }
                    }
                }
            }
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    showCreateChannel = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showCreateChannel) {
            NavigationStack {
                Form {
                    TextField("Channel Name", text: $channelName)
                    Toggle("Default Channel", isOn: $isDefault)
                }
                .navigationTitle("New PTT Channel")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showCreateChannel = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Create") {
                            Task {
                                _ = await pttVM.createChannel(groupId, name: channelName, isDefault: isDefault)
                                showCreateChannel = false
                                channelName = ""
                                isDefault = false
                            }
                        }
                        .disabled(channelName.isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }
}

private struct ActivePTTView: View {
    @Environment(PTTViewModel.self) private var pttVM
    @Environment(WebSocketService.self) private var webSocket
    let channel: PTTChannel

    var body: some View {
        VStack(spacing: 24) {
            Text(channel.name)
                .font(.title2.bold())

            Spacer()

            // Floor holder status
            if let holder = pttVM.floorHolder {
                Text("\(holder.name) is speaking")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            } else if pttVM.isTalking {
                Text("Speaking...")
                    .font(.subheadline)
                    .foregroundStyle(.green)
            } else {
                Text("Hold to talk")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            // PTT button
            Button {
                // Handled by gesture
            } label: {
                Image(systemName: "mic.fill")
                    .font(.system(size: 40))
                    .foregroundStyle(.white)
                    .frame(width: 120, height: 120)
                    .background(pttVM.isTalking ? .green : Color(.systemGray3))
                    .clipShape(Circle())
                    .shadow(color: pttVM.isTalking ? .green.opacity(0.4) : .clear, radius: 16)
            }
            .simultaneousGesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { _ in
                        guard !pttVM.isTalking else { return }
                        Task { await pttVM.startTalking(webSocket: webSocket) }
                    }
                    .onEnded { _ in
                        Task { await pttVM.stopTalking(webSocket: webSocket) }
                    }
            )

            Spacer()

            Button("Leave Channel") {
                Task { await pttVM.leaveChannel() }
            }
            .buttonStyle(.bordered)
            .tint(.red)
        }
        .padding()
    }
}
