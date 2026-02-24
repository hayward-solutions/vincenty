import Foundation
import MapLibre
import UIKit

/// Manages location marker annotations on the MLNMapView.
///
/// Mirrors the web client's `location-markers.tsx`:
/// - Creates/updates/removes `MLNPointAnnotation` for each device
/// - Uses stable per-user color assignment (cycling through fallback palette)
/// - Updates position, heading rotation, and popup content on each change
/// - Skips the current user's own device (handled by SelfMarkerController)
@MainActor
final class LocationMarkersController {

    private var mapView: MLNMapView?
    private var annotations: [String: MLNPointAnnotation] = [:]  // deviceId -> annotation
    private var userColors: [String: UIColor] = [:]  // userId -> color
    private var colorIndex = 0

    private static let fallbackColors: [UIColor] = [
        UIColor(hex: "#3b82f6"),  // blue
        UIColor(hex: "#ef4444"),  // red
        UIColor(hex: "#10b981"),  // emerald
        UIColor(hex: "#f59e0b"),  // amber
        UIColor(hex: "#8b5cf6"),  // violet
        UIColor(hex: "#ec4899"),  // pink
        UIColor(hex: "#06b6d4"),  // cyan
        UIColor(hex: "#84cc16"),  // lime
    ]

    // MARK: - Setup

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    // MARK: - Update

    /// Sync annotations with the current set of locations.
    /// `currentUserId` is excluded (rendered by SelfMarkerController).
    /// `groups` is used to resolve per-group marker colors.
    func update(
        locations: [UserLocation],
        currentUserId: String?,
        groups: [Group]
    ) {
        guard let mapView else { return }

        let groupMap = Dictionary(uniqueKeysWithValues: groups.map { ($0.id, $0) })

        // Build set of active device IDs (excluding self)
        var activeDeviceIds = Set<String>()

        for loc in locations {
            if loc.userId == currentUserId { continue }
            activeDeviceIds.insert(loc.deviceId)

            let color = resolveColor(for: loc, groupMap: groupMap)

            if let existing = annotations[loc.deviceId] {
                // Update position
                existing.coordinate = CLLocationCoordinate2D(latitude: loc.lat, longitude: loc.lng)
                existing.title = loc.displayName.isEmpty ? loc.username : loc.displayName
                existing.subtitle = formatSubtitle(loc)
            } else {
                // Create new annotation
                let annotation = MLNPointAnnotation()
                annotation.coordinate = CLLocationCoordinate2D(
                    latitude: loc.lat, longitude: loc.lng)
                annotation.title = loc.displayName.isEmpty ? loc.username : loc.displayName
                annotation.subtitle = formatSubtitle(loc)
                mapView.addAnnotation(annotation)
                annotations[loc.deviceId] = annotation
            }
        }

        // Remove stale annotations
        let staleIds = Set(annotations.keys).subtracting(activeDeviceIds)
        for id in staleIds {
            if let annotation = annotations.removeValue(forKey: id) {
                mapView.removeAnnotation(annotation)
            }
        }
    }

    /// Remove all annotations.
    func removeAll() {
        guard let mapView else { return }
        for annotation in annotations.values {
            mapView.removeAnnotation(annotation)
        }
        annotations.removeAll()
    }

    // MARK: - Private

    private func resolveColor(for loc: UserLocation, groupMap: [String: Group]) -> UIColor {
        // Check group marker color first
        if let group = groupMap[loc.groupId], !group.markerColor.isEmpty {
            return UIColor(hex: group.markerColor)
        }

        // Stable per-user color
        if let existing = userColors[loc.userId] {
            return existing
        }

        let color = Self.fallbackColors[colorIndex % Self.fallbackColors.count]
        colorIndex += 1
        userColors[loc.userId] = color
        return color
    }

    private func formatSubtitle(_ loc: UserLocation) -> String {
        var parts: [String] = []
        parts.append(loc.deviceName)
        if loc.isPrimary { parts.append("Primary") }

        let lat = String(format: "%.5f", loc.lat)
        let lng = String(format: "%.5f", loc.lng)
        parts.append("\(lat), \(lng)")

        if let speed = loc.speed, speed > 0 {
            let kmh = String(format: "%.1f", speed * 3.6)
            parts.append("\(kmh) km/h")
        }

        return parts.joined(separator: " · ")
    }
}

// MARK: - UIColor+Hex Init

private extension UIColor {
    convenience init(hex: String) {
        var hexSanitized = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        hexSanitized = hexSanitized.hasPrefix("#") ? String(hexSanitized.dropFirst()) : hexSanitized

        var rgb: UInt64 = 0
        Scanner(string: hexSanitized).scanHexInt64(&rgb)

        let r = CGFloat((rgb >> 16) & 0xFF) / 255.0
        let g = CGFloat((rgb >> 8) & 0xFF) / 255.0
        let b = CGFloat(rgb & 0xFF) / 255.0

        self.init(red: r, green: g, blue: b, alpha: 1.0)
    }
}
