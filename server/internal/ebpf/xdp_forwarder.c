//go:build ignore

// NexTunnel XDP fast path.
//
// 编译示例：
//   clang -O2 -g -target bpf -D__TARGET_ARCH_x86 -c xdp_forwarder.c -o xdp_forwarder_bpfel.o
//
// 说明：内核态仅执行 IPv4 TCP/UDP 目的端口级 PASS/DROP/REDIRECT。
// CIDR、源端口和优先级等复杂策略继续由 Go 用户态 RuleMap 处理，避免 XDP 验证器路径过度复杂。

#define SEC(NAME) __attribute__((section(NAME), used))

typedef unsigned char __u8;
typedef unsigned short __u16;
typedef unsigned int __u32;
typedef unsigned long long __u64;

#define BPF_MAP_TYPE_HASH 1
#define BPF_MAP_TYPE_ARRAY 2
#define BPF_MAP_TYPE_DEVMAP 14

#define XDP_DROP 1
#define XDP_PASS 2

#define BPF_ANY 0
#define ETH_P_IP 0x0800
#define IPPROTO_TCP 6
#define IPPROTO_UDP 17

#define ACTION_PASS 1
#define ACTION_DROP 2
#define ACTION_REDIRECT 3

#define MAX_L4_RULES 4096
#define MAX_TX_PORTS 128

#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__
#define bpf_ntohs(x) ((__u16)((((__u16)(x) & 0x00ffU) << 8) | (((__u16)(x) & 0xff00U) >> 8)))
#define bpf_htons(x) bpf_ntohs(x)
#else
#define bpf_ntohs(x) (x)
#define bpf_htons(x) (x)
#endif

struct xdp_md {
	__u32 data;
	__u32 data_end;
	__u32 data_meta;
	__u32 ingress_ifindex;
	__u32 rx_queue_index;
};

struct ethhdr {
	__u8 h_dest[6];
	__u8 h_source[6];
	__u16 h_proto;
};

struct iphdr {
	__u8 version_ihl;
	__u8 tos;
	__u16 tot_len;
	__u16 id;
	__u16 frag_off;
	__u8 ttl;
	__u8 protocol;
	__u16 check;
	__u32 saddr;
	__u32 daddr;
};

struct tcphdr_min {
	__u16 source;
	__u16 dest;
};

struct udphdr_min {
	__u16 source;
	__u16 dest;
};

struct l4_rule_key {
	__u8 protocol;
	__u8 pad;
	__u16 dst_port;
};

struct l4_rule_value {
	__u32 action;
	__u32 ifindex;
};

struct xdp_stats {
	__u64 packets_forwarded;
	__u64 bytes_forwarded;
	__u64 packets_dropped;
};

struct bpf_map_def {
	__u32 type;
	__u32 key_size;
	__u32 value_size;
	__u32 max_entries;
	__u32 map_flags;
};

struct bpf_map_def SEC("maps") l4_rules = {
	.type = BPF_MAP_TYPE_HASH,
	.key_size = sizeof(struct l4_rule_key),
	.value_size = sizeof(struct l4_rule_value),
	.max_entries = MAX_L4_RULES,
};

struct bpf_map_def SEC("maps") tx_ports = {
	.type = BPF_MAP_TYPE_DEVMAP,
	.key_size = sizeof(__u32),
	.value_size = sizeof(__u32),
	.max_entries = MAX_TX_PORTS,
};

struct bpf_map_def SEC("maps") xdp_stats_map = {
	.type = BPF_MAP_TYPE_ARRAY,
	.key_size = sizeof(__u32),
	.value_size = sizeof(struct xdp_stats),
	.max_entries = 1,
};

static void *(*bpf_map_lookup_elem)(void *map, const void *key) = (void *)1;
static long (*bpf_map_update_elem)(void *map, const void *key, const void *value, __u64 flags) = (void *)2;
static long (*bpf_redirect_map)(void *map, __u32 key, __u64 flags) = (void *)51;

static __inline void record_forwarded(__u64 packet_len) {
	__u32 key = 0;
	struct xdp_stats *stats = bpf_map_lookup_elem(&xdp_stats_map, &key);
	if (!stats) {
		return;
	}
	__sync_fetch_and_add(&stats->packets_forwarded, 1);
	__sync_fetch_and_add(&stats->bytes_forwarded, packet_len);
}

static __inline void record_dropped(void) {
	__u32 key = 0;
	struct xdp_stats *stats = bpf_map_lookup_elem(&xdp_stats_map, &key);
	if (!stats) {
		return;
	}
	__sync_fetch_and_add(&stats->packets_dropped, 1);
}

static __inline int pass_packet(__u64 packet_len) {
	record_forwarded(packet_len);
	return XDP_PASS;
}

static __inline int parse_l4_key(void *data, void *data_end, struct l4_rule_key *key) {
	struct ethhdr *eth = data;
	if ((void *)(eth + 1) > data_end) {
		return 0;
	}
	if (eth->h_proto != bpf_htons(ETH_P_IP)) {
		return 0;
	}

	struct iphdr *ip = (void *)(eth + 1);
	if ((void *)(ip + 1) > data_end) {
		return 0;
	}

	__u8 ihl = ip->version_ihl & 0x0f;
	if (ihl < 5) {
		return 0;
	}
	__u32 ip_header_len = (__u32)ihl * 4;
	if ((void *)ip + ip_header_len > data_end) {
		return 0;
	}

	key->protocol = ip->protocol;
	key->pad = 0;

	void *transport = (void *)ip + ip_header_len;
	if (ip->protocol == IPPROTO_TCP) {
		struct tcphdr_min *tcp = transport;
		if ((void *)(tcp + 1) > data_end) {
			return 0;
		}
		key->dst_port = bpf_ntohs(tcp->dest);
		return 1;
	}

	if (ip->protocol == IPPROTO_UDP) {
		struct udphdr_min *udp = transport;
		if ((void *)(udp + 1) > data_end) {
			return 0;
		}
		key->dst_port = bpf_ntohs(udp->dest);
		return 1;
	}

	return 0;
}

SEC("xdp")
int xdp_nextunnel_forward(struct xdp_md *ctx) {
	void *data = (void *)(long)ctx->data;
	void *data_end = (void *)(long)ctx->data_end;
	__u64 packet_len = data_end - data;

	struct l4_rule_key key = {};
	if (!parse_l4_key(data, data_end, &key)) {
		return pass_packet(packet_len);
	}

	struct l4_rule_value *rule = bpf_map_lookup_elem(&l4_rules, &key);
	if (!rule || rule->action == ACTION_PASS) {
		return pass_packet(packet_len);
	}

	if (rule->action == ACTION_DROP) {
		record_dropped();
		return XDP_DROP;
	}

	if (rule->action == ACTION_REDIRECT) {
		record_forwarded(packet_len);
		return bpf_redirect_map(&tx_ports, rule->ifindex, 0);
	}

	return pass_packet(packet_len);
}

char __license[] SEC("license") = "Dual MIT/GPL";
