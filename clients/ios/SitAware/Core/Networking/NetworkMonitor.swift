import Foundation
import Network

/// Observes network connectivity changes using NWPathMonitor.
@Observable @MainActor
final class NetworkMonitor {
    private(set) var isConnected: Bool = true
    private(set) var connectionType: ConnectionType = .unknown

    private let monitor = NWPathMonitor()
    private let queue = DispatchQueue(label: "com.sitaware.networkmonitor")

    enum ConnectionType: Sendable {
        case wifi
        case cellular
        case wired
        case unknown
    }

    func start() {
        monitor.pathUpdateHandler = { [weak self] path in
            Task { @MainActor [weak self] in
                guard let self else { return }
                self.isConnected = path.status == .satisfied
                self.connectionType = self.resolveType(path)
            }
        }
        monitor.start(queue: queue)
    }

    func stop() {
        monitor.cancel()
    }

    private func resolveType(_ path: NWPath) -> ConnectionType {
        if path.usesInterfaceType(.wifi) { return .wifi }
        if path.usesInterfaceType(.cellular) { return .cellular }
        if path.usesInterfaceType(.wiredEthernet) { return .wired }
        return .unknown
    }
}
