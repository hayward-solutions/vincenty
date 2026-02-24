import Foundation
import MapLibre
import UIKit

/// Manages the current user's own position marker on the map.
///
/// Mirrors the web client's `self-marker.tsx`:
/// - Adds/updates a single annotation for the user's GPS position
/// - Supports auto-centering on first position fix
/// - Removes the marker when self-location display is toggled off
@MainActor
final class SelfMarkerController {

    private var mapView: MLNMapView?
    private var annotation: MLNPointAnnotation?
    private var hasCentered = false

    /// Whether auto-centering has been performed on first fix.
    var didAutoCenter: Bool { hasCentered }

    // MARK: - Setup

    func attach(to mapView: MLNMapView) {
        self.mapView = mapView
    }

    // MARK: - Update

    /// Update the self marker position. Pass `nil` to remove.
    func update(position: (lat: Double, lng: Double, heading: Double?)?, autoCenter: Bool) {
        guard let mapView else { return }

        guard let position else {
            // Remove marker if position is nil (e.g. "Show self" toggled off)
            removeMarker()
            return
        }

        let coord = CLLocationCoordinate2D(latitude: position.lat, longitude: position.lng)

        if let existing = annotation {
            // Update position
            existing.coordinate = coord
            existing.subtitle = String(
                format: "%.5f, %.5f", position.lat, position.lng)
        } else {
            // Create annotation
            let ann = MLNPointAnnotation()
            ann.coordinate = coord
            ann.title = "You"
            ann.subtitle = String(format: "%.5f, %.5f", position.lat, position.lng)
            mapView.addAnnotation(ann)
            self.annotation = ann
        }

        // Auto-center on first fix
        if autoCenter && !hasCentered {
            hasCentered = true
            let zoom = max(mapView.zoomLevel, 14)
            let camera = MLNMapCamera(
                lookingAtCenter: coord,
                altitude: mapView.camera.altitude,
                pitch: mapView.camera.pitch,
                heading: mapView.direction)
            mapView.setZoomLevel(zoom, animated: false)
            mapView.fly(to: camera, withDuration: 1.5, completionHandler: nil)
        }
    }

    /// Remove the self marker.
    func removeMarker() {
        guard let mapView, let annotation else { return }
        mapView.removeAnnotation(annotation)
        self.annotation = nil
    }

    /// Reset the auto-center flag (e.g. when user logs out).
    func resetAutoCenter() {
        hasCentered = false
    }
}
