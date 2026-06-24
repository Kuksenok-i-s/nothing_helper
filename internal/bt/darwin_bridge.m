#import <Foundation/Foundation.h>
#import <IOBluetooth/IOBluetooth.h>
#import "darwin_bridge.h"

static NSString *NothingSPPUUID = @"AEAC4A03-DFF5-498F-843A-34487CF133EB";
static const NSTimeInterval kOpenTimeoutSec = 1.0;
static const NSTimeInterval kConnectTimeoutSec = 1.0;
static const NSTimeInterval kCloseDrainSec = 0.5;
static const NSTimeInterval kOpenRetryDrainSec = 1.0;
static const int kOpenMaxAttempts = 3;
// bt_transport_read blocks on NSCondition until RFCOMM data arrives. Do not pump
// CFRunLoopRunInMode from Go reader threads: sample(1) showed heavy CPU in
// __CFRunLoopCopyMode when that was used for idle polling.
static const NSTimeInterval kReadWaitSliceSec = 1.0;

@interface BTRFCOMMDelegate : NSObject <IOBluetoothRFCOMMChannelDelegate>
@property (nonatomic, assign) int handle;
@property (nonatomic, assign) IOReturn openStatus;
@property (nonatomic, assign) BOOL openDone;
@property (nonatomic, assign) BOOL closeDone;
@end

static NSLock *gLock;
static NSLock *gOpenLock;
static NSMutableDictionary<NSNumber *, IOBluetoothRFCOMMChannel *> *gChannels;
static NSMutableDictionary<NSNumber *, NSMutableData *> *gReadBuffers;
static NSMutableDictionary<NSNumber *, NSCondition *> *gReadConds;
static NSMutableDictionary<NSNumber *, BTRFCOMMDelegate *> *gDelegates;
static int gNextHandle = 1;
static BOOL gInitialized = NO;

@implementation BTRFCOMMDelegate

- (void)rfcommChannelData:(IOBluetoothRFCOMMChannel *)rfcommChannel data:(void *)dataPointer length:(size_t)dataLength {
    (void)rfcommChannel;
    NSData *chunk = [NSData dataWithBytes:dataPointer length:dataLength];
    NSCondition *cond = nil;
    [gLock lock];
    NSMutableData *buf = gReadBuffers[@(self.handle)];
    if (buf) {
        [buf appendData:chunk];
    }
    cond = gReadConds[@(self.handle)];
    [gLock unlock];
    if (cond) {
        [cond lock];
        [cond signal];
        [cond unlock];
    }
}

- (void)rfcommChannelClosed:(IOBluetoothRFCOMMChannel *)rfcommChannel {
    (void)rfcommChannel;
    self.closeDone = YES;
}

- (void)rfcommChannelOpenComplete:(IOBluetoothRFCOMMChannel *)rfcommChannel status:(IOReturn)error {
    (void)rfcommChannel;
    self.openStatus = error;
    self.openDone = YES;
}

@end

static void ensureInit(void) {
    if (gInitialized) {
        return;
    }
    gLock = [[NSLock alloc] init];
    gOpenLock = [[NSLock alloc] init];
    gChannels = [NSMutableDictionary dictionary];
    gReadBuffers = [NSMutableDictionary dictionary];
    gReadConds = [NSMutableDictionary dictionary];
    gDelegates = [NSMutableDictionary dictionary];
    gInitialized = YES;
}

static void btRunSync(void (^block)(void)) {
    ensureInit();
    @autoreleasepool {
        block();
    }
}

static int serviceHasSPPUUID(IOBluetoothDevice *dev) {
    for (IOBluetoothSDPServiceRecord *rec in dev.services) {
        NSString *desc = [rec description];
        if (desc && [desc rangeOfString:NothingSPPUUID options:NSCaseInsensitiveSearch].location != NSNotFound) {
            return 1;
        }
    }
    return 0;
}

@interface BTSDPQueryRunner : NSObject
@property (nonatomic, assign) IOReturn status;
@property (nonatomic, assign) BOOL done;
@end

@implementation BTSDPQueryRunner

- (void)sdpQueryComplete:(IOBluetoothDevice *)device status:(IOReturn)status {
    (void)device;
    self.status = status;
    self.done = YES;
}

@end

static IOBluetoothSDPUUID *nothingSPPUUID(void) {
    static IOBluetoothSDPUUID *uuid = nil;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        const unsigned char bytes[] = {
            0xAE, 0xAC, 0x4A, 0x03, 0xDF, 0xF5, 0x49, 0x8F,
            0x84, 0x3A, 0x34, 0x48, 0x7C, 0xF1, 0x33, 0xEB,
        };
        uuid = [[IOBluetoothSDPUUID alloc] initWithBytes:bytes length:sizeof(bytes)];
    });
    return uuid;
}

static int rfcommChannelFromRecord(IOBluetoothSDPServiceRecord *rec) {
    if (!rec) {
        return 0;
    }
    BluetoothRFCOMMChannelID channelID = 0;
    if ([rec getRFCOMMChannelID:&channelID] != kIOReturnSuccess) {
        return 0;
    }
    return (int)channelID;
}

static int ensureSDPRecords(IOBluetoothDevice *dev) {
    if ([dev.services count] > 0) {
        return 0;
    }
    BTSDPQueryRunner *runner = [[BTSDPQueryRunner alloc] init];
    IOReturn err = [dev performSDPQuery:runner];
    if (err != kIOReturnSuccess) {
        return -6;
    }
    NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:5.0];
    while (!runner.done && [deadline timeIntervalSinceNow] > 0) {
        CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.05, YES);
    }
    if (!runner.done) {
        return -6;
    }
    if (runner.status != kIOReturnSuccess) {
        return -6;
    }
    return 0;
}

static int resolveRFCOMMChannelForDevice(IOBluetoothDevice *dev, int hint, int *out) {
    if (!dev || !out) {
        return -1;
    }
    if (ensureSDPRecords(dev) != 0 && [dev.services count] == 0) {
        if (hint >= 1 && hint <= 63) {
            *out = hint;
            return 0;
        }
        return -6;
    }
    IOBluetoothSDPServiceRecord *rec = [dev getServiceRecordForUUID:nothingSPPUUID()];
    int ch = rfcommChannelFromRecord(rec);
    if (ch >= 1 && ch <= 63) {
        *out = ch;
        return 0;
    }
    IOBluetoothSDPUUID *serialUUID = [IOBluetoothSDPUUID uuid16:kBluetoothSDPUUID16ServiceClassSerialPort];
    rec = [dev getServiceRecordForUUID:serialUUID];
    ch = rfcommChannelFromRecord(rec);
    if (ch >= 1 && ch <= 63) {
        *out = ch;
        return 0;
    }
    for (IOBluetoothSDPServiceRecord *svc in dev.services) {
        NSString *desc = [svc description];
        if (desc && [desc rangeOfString:NothingSPPUUID options:NSCaseInsensitiveSearch].location != NSNotFound) {
            ch = rfcommChannelFromRecord(svc);
            if (ch >= 1 && ch <= 63) {
                *out = ch;
                return 0;
            }
        }
    }
    if (hint >= 1 && hint <= 63) {
        *out = hint;
        return 0;
    }
    return -6;
}

static IOBluetoothDevice *deviceForMAC(NSString *mac);

int bt_resolve_rfcomm_channel(const char *mac, int hint, int *out_channel) {
    __block int rc = -1;
    btRunSync(^{
        if (!mac || !out_channel) {
            rc = -1;
            return;
        }
        IOBluetoothDevice *dev = deviceForMAC([NSString stringWithUTF8String:mac]);
        if (!dev) {
            rc = -2;
            return;
        }
        rc = resolveRFCOMMChannelForDevice(dev, hint, out_channel);
    });
    return rc;
}

int bt_init(void) {
    ensureInit();
    return 0;
}

void bt_shutdown(void) {
    btRunSync(^{
        [gLock lock];
        for (NSNumber *key in [gChannels copy]) {
            IOBluetoothRFCOMMChannel *ch = gChannels[key];
            [ch closeChannel];
        }
        [gChannels removeAllObjects];
        [gReadBuffers removeAllObjects];
        [gReadConds removeAllObjects];
        [gDelegates removeAllObjects];
        [gLock unlock];
    });
}

static NSString *normalizeMACString(NSString *mac) {
    if (!mac) {
        return @"";
    }
    return [[mac stringByReplacingOccurrencesOfString:@"-" withString:@":"] uppercaseString];
}

static IOBluetoothDevice *deviceForMAC(NSString *mac) {
    NSString *norm = normalizeMACString(mac);
    IOBluetoothDevice *target = [IOBluetoothDevice deviceWithAddressString:norm];
    if (!target) {
        target = [IOBluetoothDevice deviceWithAddressString:[norm stringByReplacingOccurrencesOfString:@":" withString:@"-"]];
    }
    if (target) {
        return target;
    }
    NSArray *paired = [IOBluetoothDevice pairedDevices];
    for (IOBluetoothDevice *dev in paired) {
        if ([normalizeMACString(dev.addressString) isEqualToString:norm]) {
            return dev;
        }
    }
    return nil;
}

static void fillDeviceRecord(IOBluetoothDevice *dev, bt_device_record *out) {
    memset(out, 0, sizeof(*out));
    NSString *mac = normalizeMACString(dev.addressString);
    strncpy(out->mac, [mac UTF8String], BT_MAC_LEN - 1);
    NSString *name = dev.name ?: dev.addressString;
    strncpy(out->name, [name UTF8String], BT_NAME_LEN - 1);
    out->connected = dev.isConnected ? 1 : 0;
    out->paired = dev.isPaired ? 1 : 0;
    out->spp = serviceHasSPPUUID(dev);
    NSMutableString *info = [NSMutableString string];
    [info appendFormat:@"Name: %@\n", name ?: @""];
    [info appendFormat:@"Connected: %@\n", dev.isConnected ? @"yes" : @"no"];
    [info appendFormat:@"Paired: %@\n", dev.isPaired ? @"yes" : @"no"];
    if (out->spp) {
        [info appendFormat:@"UUID: %@\n", NothingSPPUUID];
    }
    strncpy(out->info, [info UTF8String], BT_INFO_LEN - 1);
}

static int waitForConnected(IOBluetoothDevice *dev, NSTimeInterval timeoutSec) {
    NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:timeoutSec];
    while (!dev.isConnected && [deadline timeIntervalSinceNow] > 0) {
        [NSThread sleepForTimeInterval:0.05];
    }
    return dev.isConnected ? 0 : -1;
}

static void drainRunLoop(NSTimeInterval seconds) {
    NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:seconds];
    while ([deadline timeIntervalSinceNow] > 0) {
        CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.01, YES);
    }
}

static void closeTrackedChannelsForDevice(IOBluetoothDevice *dev) {
    if (!dev) {
        return;
    }
    NSString *norm = normalizeMACString(dev.addressString);
    [gLock lock];
    NSArray *keys = [gChannels copy];
    for (NSNumber *key in keys) {
        IOBluetoothRFCOMMChannel *ch = gChannels[key];
        if (!ch) {
            continue;
        }
        IOBluetoothDevice *owner = [ch getDevice];
        if (owner && [normalizeMACString(owner.addressString) isEqualToString:norm]) {
            [ch closeChannel];
            drainRunLoop(kCloseDrainSec);
            [gChannels removeObjectForKey:key];
            [gReadBuffers removeObjectForKey:key];
            [gDelegates removeObjectForKey:key];
        }
    }
    [gLock unlock];
}

int bt_discover_paired(bt_device_record *out, int max, int *count) {
    __block int rc = 0;
    btRunSync(^{
        if (!out || !count || max <= 0) {
            rc = -1;
            return;
        }
        *count = 0;
        NSArray *paired = [IOBluetoothDevice pairedDevices];
        for (IOBluetoothDevice *dev in paired) {
            if (*count >= max) {
                break;
            }
            bt_device_record rec;
            fillDeviceRecord(dev, &rec);
            out[*count] = rec;
            (*count)++;
        }
    });
    return rc;
}

int bt_device_has_spp(const char *mac) {
    __block int rc = 0;
    btRunSync(^{
        IOBluetoothDevice *dev = deviceForMAC([NSString stringWithUTF8String:mac]);
        if (!dev) {
            rc = 0;
            return;
        }
        rc = serviceHasSPPUUID(dev);
    });
    return rc;
}

int bt_is_connected(const char *mac) {
    __block int rc = 0;
    btRunSync(^{
        IOBluetoothDevice *dev = deviceForMAC([NSString stringWithUTF8String:mac]);
        rc = (dev && dev.isConnected) ? 1 : 0;
    });
    return rc;
}

char *bt_device_info(const char *mac) {
    __block char *result = strdup("");
    btRunSync(^{
        IOBluetoothDevice *dev = deviceForMAC([NSString stringWithUTF8String:mac]);
        if (!dev) {
            return;
        }
        bt_device_record rec;
        fillDeviceRecord(dev, &rec);
        free(result);
        result = strdup(rec.info);
    });
    return result;
}

static int ensureDeviceConnected(IOBluetoothDevice *dev) {
    if (dev.isConnected) {
        return 0;
    }
    if (![dev openConnection]) {
        return -3;
    }
    return waitForConnected(dev, kConnectTimeoutSec) == 0 ? 0 : -3;
}

static int refreshDeviceConnection(IOBluetoothDevice *dev) {
    if (dev.isConnected) {
        [dev closeConnection];
        drainRunLoop(kOpenRetryDrainSec);
    }
    return ensureDeviceConnected(dev);
}

static int tryOpenRFCOMMChannel(IOBluetoothDevice *dev, int channel, IOBluetoothRFCOMMChannel **outChannel,
                               BTRFCOMMDelegate **outDelegate) {
    IOBluetoothRFCOMMChannel *rfcomm = nil;
    BTRFCOMMDelegate *delegate = nil;
    for (int attempt = 0; attempt < kOpenMaxAttempts; attempt++) {
        delegate = [[BTRFCOMMDelegate alloc] init];
        rfcomm = nil;
        (void)[dev openRFCOMMChannelSync:&rfcomm
                          withChannelID:(BluetoothRFCOMMChannelID)channel
                               delegate:delegate];
        if (rfcomm && rfcomm.isOpen) {
            *outChannel = rfcomm;
            *outDelegate = delegate;
            return 0;
        }
        if (rfcomm) {
            [rfcomm closeChannel];
            drainRunLoop(kOpenRetryDrainSec);
        } else if (attempt + 1 < kOpenMaxAttempts) {
            drainRunLoop(kOpenRetryDrainSec);
        }
    }
    return -5;
}

int bt_open_rfcomm(const char *mac, int channel, bt_transport_handle *out) {
    ensureInit();
    [gOpenLock lock];
    int rc = -1;
    @autoreleasepool {
        if (!mac || !out || channel < 1 || channel > 63) {
            rc = -1;
        } else {
            IOBluetoothDevice *dev = deviceForMAC([NSString stringWithUTF8String:mac]);
            if (!dev) {
                rc = -2;
            } else {
                closeTrackedChannelsForDevice(dev);
                drainRunLoop(kCloseDrainSec);
                rc = ensureDeviceConnected(dev);
                if (rc == 0) {
                    IOBluetoothRFCOMMChannel *rfcomm = nil;
                    BTRFCOMMDelegate *delegate = nil;
                    rc = tryOpenRFCOMMChannel(dev, channel, &rfcomm, &delegate);
                    if (rc != 0) {
                        if (refreshDeviceConnection(dev) == 0) {
                            closeTrackedChannelsForDevice(dev);
                            drainRunLoop(kCloseDrainSec);
                            rc = tryOpenRFCOMMChannel(dev, channel, &rfcomm, &delegate);
                        }
                    }
                    if (rc == 0 && rfcomm && delegate) {
                        [rfcomm setDelegate:delegate];
                        [gLock lock];
                        int handle = gNextHandle++;
                        delegate.handle = handle;
                        gDelegates[@(handle)] = delegate;
                        gChannels[@(handle)] = rfcomm;
                        gReadBuffers[@(handle)] = [NSMutableData data];
                        gReadConds[@(handle)] = [[NSCondition alloc] init];
                        [gLock unlock];
                        memset(out, 0, sizeof(*out));
                        out->handle = handle;
                        strncpy(out->mac, mac, BT_MAC_LEN - 1);
                        out->channel = channel;
                        rc = 0;
                    } else if (rc == 0) {
                        rc = -5;
                    }
                }
            }
        }
    }
    [gOpenLock unlock];
    return rc;
}

int bt_transport_read(int handle, uint8_t *buf, int buflen, int timeout_ms) {
    if (!buf || buflen <= 0) {
        return -1;
    }
    NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:timeout_ms / 1000.0];
    for (;;) {
        [gLock lock];
        NSMutableData *data = gReadBuffers[@(handle)];
        NSCondition *cond = gReadConds[@(handle)];
        if (!data || !cond) {
            [gLock unlock];
            return -1;
        }
        if (data.length > 0) {
            NSUInteger n = MIN((NSUInteger)buflen, data.length);
            [data getBytes:buf length:n];
            [data replaceBytesInRange:NSMakeRange(0, n) withBytes:NULL length:0];
            [gLock unlock];
            return (int)n;
        }
        [cond lock];
        [gLock unlock];

        NSTimeInterval remaining = [deadline timeIntervalSinceNow];
        if (remaining <= 0) {
            [cond unlock];
            return 0;
        }
        NSTimeInterval slice = remaining < kReadWaitSliceSec ? remaining : kReadWaitSliceSec;
        [cond waitUntilDate:[NSDate dateWithTimeIntervalSinceNow:slice]];
        [cond unlock];
    }
}

int bt_transport_write(int handle, const uint8_t *data, int len) {
    if (!data || len <= 0) {
        return -1;
    }
    ensureInit();
    [gLock lock];
    IOBluetoothRFCOMMChannel *ch = gChannels[@(handle)];
    [gLock unlock];
    if (!ch) {
        return -2;
    }
    IOReturn status = [ch writeSync:(void *)data length:(UInt16)len];
    if (status != kIOReturnSuccess) {
        return -3;
    }
    return len;
}

int bt_transport_close(int handle) {
    ensureInit();
    [gOpenLock lock];
    int result = -1;
    @autoreleasepool {
        [gLock lock];
        IOBluetoothRFCOMMChannel *ch = gChannels[@(handle)];
        BTRFCOMMDelegate *delegate = gDelegates[@(handle)];
        NSCondition *cond = gReadConds[@(handle)];
        [gChannels removeObjectForKey:@(handle)];
        [gReadBuffers removeObjectForKey:@(handle)];
        [gReadConds removeObjectForKey:@(handle)];
        [gDelegates removeObjectForKey:@(handle)];
        [gLock unlock];
        if (cond) {
            [cond lock];
            [cond broadcast];
            [cond unlock];
        }
        if (ch) {
            if (delegate) {
                delegate.closeDone = NO;
            }
            [ch closeChannel];
            if (delegate) {
                NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:kCloseDrainSec];
                while (!delegate.closeDone && [deadline timeIntervalSinceNow] > 0) {
                    drainRunLoop(0.01);
                }
            }
            result = 0;
        }
    }
    [gOpenLock unlock];
    return result;
}

char *bt_host_adapter_mac(void) {
    __block char *result = strdup("");
    btRunSync(^{
        IOBluetoothHostController *host = [IOBluetoothHostController defaultController];
        if (host && host.addressAsString.length > 0) {
            free(result);
            result = strdup([normalizeMACString(host.addressAsString) UTF8String]);
        }
    });
    return result;
}
