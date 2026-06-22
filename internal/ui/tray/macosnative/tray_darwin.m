#import <Cocoa/Cocoa.h>
#include <stdint.h>

@class TrayMenuHandler;

extern void tray_go_menu_click(int32_t tag);

static NSStatusItem *statusItem = nil;
static NSMenuItem *statusMenuItem = nil;
static NSMenuItem *batteryMenuItem = nil;
static TrayMenuHandler *menuHandler = nil;

@interface TrayMenuHandler : NSObject
- (void)handleMenu:(id)sender;
@end

@implementation TrayMenuHandler
- (void)handleMenu:(id)sender {
  NSMenuItem *item = (NSMenuItem *)sender;
  tray_go_menu_click((int32_t)[item tag]);
}
@end

static NSMenuItem *add_action_item(NSMenu *menu, NSString *title, int tag) {
  NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title
                                                action:@selector(handleMenu:)
                                         keyEquivalent:@""];
  [item setTarget:menuHandler];
  [item setTag:tag];
  [menu addItem:item];
  return item;
}

void tray_darwin_schedule_init(const char *tooltip, const void *iconData, int iconLen,
                               int showWindow, int showReconnect) {
  NSString *tooltipStr = tooltip ? @(tooltip) : @"tws_manager";
  NSData *iconCopy = nil;
  if (iconData != NULL && iconLen > 0) {
    iconCopy = [NSData dataWithBytes:iconData length:iconLen];
  }

  dispatch_async(dispatch_get_main_queue(), ^{
    if (statusItem != nil) {
      return;
    }
    menuHandler = [[TrayMenuHandler alloc] init];
    statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    NSMenu *menu = [[NSMenu alloc] init];
    [menu setAutoenablesItems:NO];

    statusMenuItem = [[NSMenuItem alloc] initWithTitle:@"Disconnected"
                                              action:nil
                                       keyEquivalent:@""];
    [statusMenuItem setEnabled:NO];
    [menu addItem:statusMenuItem];

    batteryMenuItem = [[NSMenuItem alloc] initWithTitle:@"Battery: n/a"
                                                 action:nil
                                          keyEquivalent:@""];
    [batteryMenuItem setEnabled:NO];
    [menu addItem:batteryMenuItem];

    [menu addItem:[NSMenuItem separatorItem]];

    if (showWindow) {
      add_action_item(menu, @"Show window", 1);
    }
    add_action_item(menu, @"Refresh battery", 2);
    if (showReconnect) {
      add_action_item(menu, @"Reconnect", 3);
    }
    add_action_item(menu, @"Disconnect", 4);
    [menu addItem:[NSMenuItem separatorItem]];
    add_action_item(menu, @"Quit", 5);

    [statusItem setMenu:menu];
    statusItem.button.toolTip = tooltipStr;
    if (iconCopy != nil) {
      NSImage *image = [[NSImage alloc] initWithData:iconCopy];
      if (image != nil) {
        [image setSize:NSMakeSize(16, 16)];
        image.template = YES;
        statusItem.button.image = image;
      }
    }
  });
}

void tray_darwin_set_status(const char *title) {
  if (title == NULL) {
    return;
  }
  NSString *s = @(title);
  dispatch_async(dispatch_get_main_queue(), ^{
    if (statusMenuItem != nil) {
      statusMenuItem.title = s;
    }
  });
}

void tray_darwin_set_battery(const char *title) {
  if (title == NULL) {
    return;
  }
  NSString *s = @(title);
  dispatch_async(dispatch_get_main_queue(), ^{
    if (batteryMenuItem != nil) {
      batteryMenuItem.title = s;
    }
  });
}

void tray_darwin_set_tooltip(const char *tooltip) {
  if (tooltip == NULL) {
    return;
  }
  NSString *s = @(tooltip);
  dispatch_async(dispatch_get_main_queue(), ^{
    if (statusItem != nil) {
      statusItem.button.toolTip = s;
    }
  });
}
