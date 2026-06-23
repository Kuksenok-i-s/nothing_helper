#ifndef TWS_NOTIFY_DARWIN_H
#define TWS_NOTIFY_DARWIN_H

// notify_darwin_available reports whether native NSUserNotification delivery is
// usable for the current process. It returns 1 only when the process runs from a
// real application bundle (a bundle identifier is present); bare CLI binaries
// return 0 so the caller can fall back to osascript.
int notify_darwin_available(void);

// notify_darwin_post delivers a banner attributed to the host app bundle.
// Strings are UTF-8; subtitle may be empty to omit it. Safe to call from any
// goroutine: delivery is dispatched onto the main queue.
void notify_darwin_post(const char *title, const char *subtitle, const char *body);

#endif // TWS_NOTIFY_DARWIN_H
