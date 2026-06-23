#import <Foundation/Foundation.h>
#import "notify_darwin.h"

// NSUserNotification is deprecated since macOS 10.14 but remains functional and,
// unlike UNUserNotificationCenter, works reliably for ad-hoc-signed bundles
// without an async authorization round-trip. Silence the deprecation noise.
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"

@interface TWSNotifDelegate : NSObject <NSUserNotificationCenterDelegate>
@end

@implementation TWSNotifDelegate
// Force banners to appear even when our app happens to be the active app
// (otherwise macOS silently drops them for a foreground app).
- (BOOL)userNotificationCenter:(NSUserNotificationCenter *)center
     shouldPresentNotification:(NSUserNotification *)notification {
    (void)center;
    (void)notification;
    return YES;
}
@end

static TWSNotifDelegate *gNotifDelegate = nil;

int notify_darwin_available(void) {
    return [[NSBundle mainBundle] bundleIdentifier] != nil ? 1 : 0;
}

void notify_darwin_post(const char *title, const char *subtitle, const char *body) {
    // Convert C strings to NSString synchronously: the caller frees them right
    // after this returns, so they must not be touched inside the async block.
    NSString *t = title ? [NSString stringWithUTF8String:title] : @"";
    NSString *s = (subtitle && subtitle[0] != '\0') ? [NSString stringWithUTF8String:subtitle] : nil;
    NSString *b = body ? [NSString stringWithUTF8String:body] : @"";

    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSUserNotificationCenter *center = [NSUserNotificationCenter defaultUserNotificationCenter];
            if (center == nil) {
                return;
            }
            if (gNotifDelegate == nil) {
                gNotifDelegate = [[TWSNotifDelegate alloc] init];
                center.delegate = gNotifDelegate;
            }
            NSUserNotification *n = [[NSUserNotification alloc] init];
            n.title = t;
            if (s != nil) {
                n.subtitle = s;
            }
            n.informativeText = b;
            [center deliverNotification:n];
        }
    });
}

#pragma clang diagnostic pop
