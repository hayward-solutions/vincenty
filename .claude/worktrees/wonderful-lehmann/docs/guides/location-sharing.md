# Location Sharing

Vincenty tracks team members' positions in real-time on the map. This guide explains how location sharing works and how to manage it.

## How Location Sharing Works

1. When you open the **Map** page, Vincenty requests access to your browser's Geolocation API.
2. If you grant permission, your position is sent via WebSocket to the Vincenty API.
3. The API records your location in the `location_history` table and broadcasts it to all group members with read permission.
4. Other users see your marker move on their map in real-time.

## Enabling Location Sharing

### Browser Permission

The first time you visit the Map page, your browser will ask for location permission. You must allow this for location sharing to work.

- **Chrome/Edge**: Click "Allow" in the permission popup
- **Firefox**: Click "Allow Location Access"
- **Safari**: Click "Allow" or "Allow While Using App"

If you previously denied permission, you can re-enable it in your browser's site settings.

### Track My Location Button

On the map, click the **crosshair icon** (Track my location) in the right toolbar to:
- Center the map on your current position
- Begin actively sharing your location

## Location Update Frequency

Location updates are throttled to prevent excessive network traffic. The default interval is **1 second** between updates, configurable via the `WS_LOCATION_THROTTLE` environment variable.

When your position changes, the browser's Geolocation API provides:
- **Latitude and longitude**
- **Altitude** (if available)
- **Heading** (compass direction, if moving)
- **Speed** (if moving)
- **Accuracy** (estimated error radius in meters)

## Who Can See Your Location

Your location is visible to:
- **Group members** with read permission in any group you share
- **Administrators** who can see all user locations across all groups

You can only see locations of users who are in the same group(s) as you (unless you are an admin).

## Location History

Every location update is stored in the database with a timestamp. This history enables:

### Replay
Open the **Replay** panel on the map to play back location history for any time range. See [Using the Map > Replaying Location History](map.md#replaying-location-history).

### GPX Export
Export your own location history as a GPX file:
1. Open the Replay panel
2. Select a time range
3. Click **Export GPX**

The GPX file can be opened in mapping applications like Google Earth, QGIS, or Garmin BaseCamp.

### Activity Log
Your location data is also recorded in the audit trail when you perform API actions, providing a spatial context for your activity.

## Privacy Considerations

- Location sharing only works while you have the Map page open and the browser tab is active
- Closing the browser tab stops location sharing
- Your location is only visible to members of your groups (not to all users on the system, unless the viewer is an admin)
- Location history is retained indefinitely -- contact your administrator about data retention policies
- The `WS_LOCATION_THROTTLE` setting controls how frequently updates are sent (minimum 1-second interval by default)

## Troubleshooting

| Issue | Solution |
|---|---|
| No location marker shown for me | Check that you granted browser location permission. Look for a location icon in your browser's address bar. |
| Location seems inaccurate | GPS accuracy depends on your device. Desktop browsers using Wi-Fi may be less accurate than phones with GPS. |
| Location not updating | Make sure the Map page is open and the browser tab is in the foreground. Check the connection status indicator. |
| Others can't see my location | Verify you are a member of a shared group with write permission. |
