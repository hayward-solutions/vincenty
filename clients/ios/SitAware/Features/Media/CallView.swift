import SwiftUI
import LiveKit

struct CallScreen: View {
    @Environment(CallViewModel.self) private var callVM
    @Environment(WebSocketService.self) private var webSocket

    var body: some View {
        Group {
            if callVM.isInCall, let room = callVM.currentRoom {
                ActiveCallView(room: room, mediaRoom: callVM.currentMediaRoom)
            } else {
                CallListView()
            }
        }
        .navigationTitle("Calls")
        .task {
            await callVM.fetchActiveCalls()
            callVM.subscribeToCallEvents(webSocket: webSocket)
        }
        .onDisappear {
            callVM.unsubscribe()
        }
    }
}

private struct CallListView: View {
    @Environment(CallViewModel.self) private var callVM
    @State private var showNewCall = false
    @State private var callName = ""
    @State private var videoEnabled = true

    var body: some View {
        List {
            Section("Active Calls") {
                if callVM.activeCalls.isEmpty {
                    Text("No active calls")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(callVM.activeCalls) { call in
                        HStack {
                            VStack(alignment: .leading) {
                                Text(call.name)
                                    .font(.headline)
                                Text(call.isActive ? "Active" : "Ended")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Button("Join") {
                                Task { await callVM.joinCall(call.id) }
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.green)
                        }
                    }
                }
            }
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    showNewCall = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showNewCall) {
            NavigationStack {
                Form {
                    TextField("Call Name", text: $callName)
                    Toggle("Video", isOn: $videoEnabled)
                }
                .navigationTitle("New Call")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showNewCall = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Start") {
                            Task {
                                _ = await callVM.createCall(name: callName.isEmpty ? nil : callName, videoEnabled: videoEnabled)
                                showNewCall = false
                                callName = ""
                            }
                        }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .overlay {
            if callVM.isLoading {
                ProgressView()
            }
        }
    }
}

private struct ActiveCallView: View {
    @Environment(CallViewModel.self) private var callVM
    let room: Room
    let mediaRoom: MediaRoom?

    var body: some View {
        VStack(spacing: 16) {
            Text(mediaRoom?.name ?? "Call")
                .font(.title2.bold())

            Text("\(room.remoteParticipants.count + 1) participant\(room.remoteParticipants.count == 0 ? "" : "s")")
                .foregroundStyle(.secondary)

            if callVM.isVideoEnabled {
                // Video participants grid
                ScrollView {
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                        // Local video
                        if let track = room.localParticipant.firstCameraVideoTrack {
                            VideoView(track)
                                .aspectRatio(16/9, contentMode: .fit)
                                .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                        // Remote videos
                        ForEach(Array(room.remoteParticipants.values), id: \.identity) { participant in
                            if let track = participant.firstCameraVideoTrack {
                                VideoView(track)
                                    .aspectRatio(16/9, contentMode: .fit)
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                        }
                    }
                    .padding()
                }
            } else {
                // Audio-only: show participant list
                ScrollView {
                    VStack(spacing: 8) {
                        ForEach(Array(room.remoteParticipants.values), id: \.identity) { participant in
                            HStack {
                                Image(systemName: "person.circle.fill")
                                    .font(.title2)
                                Text(participant.name ?? participant.identity?.stringValue ?? "Unknown")
                                Spacer()
                            }
                            .padding(.horizontal)
                        }
                    }
                }
            }

            Spacer()

            // Call controls
            HStack(spacing: 24) {
                Button {
                    Task { await callVM.toggleMicrophone() }
                } label: {
                    Image(systemName: room.localParticipant.isMicrophoneEnabled() ? "mic.fill" : "mic.slash.fill")
                        .font(.title2)
                        .frame(width: 56, height: 56)
                        .background(room.localParticipant.isMicrophoneEnabled() ? Color(.systemGray5) : .red)
                        .clipShape(Circle())
                }

                if callVM.isVideoEnabled {
                    Button {
                        Task { await callVM.toggleCamera() }
                    } label: {
                        Image(systemName: room.localParticipant.isCameraEnabled() ? "video.fill" : "video.slash.fill")
                            .font(.title2)
                            .frame(width: 56, height: 56)
                            .background(room.localParticipant.isCameraEnabled() ? Color(.systemGray5) : .red)
                            .clipShape(Circle())
                    }
                }

                Button {
                    Task { await callVM.leaveCall() }
                } label: {
                    Image(systemName: "phone.down.fill")
                        .font(.title2)
                        .foregroundStyle(.white)
                        .frame(width: 56, height: 56)
                        .background(.red)
                        .clipShape(Circle())
                }
            }
            .padding(.bottom, 24)
        }
    }
}
