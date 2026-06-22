#ifndef DARWIN_BT_BRIDGE_H
#define DARWIN_BT_BRIDGE_H

#include <stdint.h>

#define BT_MAX_DEVICES 64
#define BT_NAME_LEN 128
#define BT_MAC_LEN 18
#define BT_INFO_LEN 512

typedef struct {
    char mac[BT_MAC_LEN];
    char name[BT_NAME_LEN];
    char alias[BT_NAME_LEN];
    char info[BT_INFO_LEN];
    int connected;
    int paired;
    int spp;
} bt_device_record;

typedef struct {
    int handle;
    char mac[BT_MAC_LEN];
    int channel;
} bt_transport_handle;

int bt_init(void);
void bt_shutdown(void);

int bt_discover_paired(bt_device_record *out, int max, int *count);
int bt_device_has_spp(const char *mac);
int bt_is_connected(const char *mac);
char *bt_device_info(const char *mac);

// Resolve RFCOMM channel via SDP (Nothing SPP UUID, then serial-port 0x1101, then hint).
// Returns 0 on success; -2 device not found; -6 SDP/channel unavailable.
int bt_resolve_rfcomm_channel(const char *mac, int hint, int *out_channel);

int bt_open_rfcomm(const char *mac, int channel, bt_transport_handle *out);
int bt_transport_read(int handle, uint8_t *buf, int buflen, int timeout_ms);
int bt_transport_write(int handle, const uint8_t *data, int len);
int bt_transport_close(int handle);

char *bt_host_adapter_mac(void);

#endif
