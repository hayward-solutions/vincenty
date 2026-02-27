import SwiftUI
import SwiftData

@main
struct SitAwareApp: App {
    @State private var authManager = AuthManager()
    @State private var networkMonitor = NetworkMonitor()
    @State private var deviceManager = DeviceManager()
    @State private var webSocketService = WebSocketService()
    @State private var locationManager = LocationSharingManager()
    @State private var syncManager = SyncManager()

    private let modelContainer: ModelContainer

    init() {
        do {
            modelContainer = try DataContainer.create()
        } catch {
            fatalError("Failed to create SwiftData container: \(error)")
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(authManager)
                .environment(networkMonitor)
                .environment(deviceManager)
                .environment(webSocketService)
                .environment(locationManager)
                .environment(syncManager)
                .modelContainer(modelContainer)
                .task {
                    AppLogger.shared.log(.info, .app, "App launched")
                    syncManager.configure(container: modelContainer)
                    networkMonitor.start()
                    await authManager.bootstrap()
                }
        }
    }
}
