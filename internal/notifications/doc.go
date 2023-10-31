// Package notifications contains code that facilitates real-time notifications:
// whenever a 'flow' record is created or updated in the database, we respond by sending
// an event to all connected clients that are authenticated as the affected user
package notifications
