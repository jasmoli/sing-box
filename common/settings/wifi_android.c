#include <errno.h>
#include <net/if.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

#include <linux/genetlink.h>
#include <linux/netlink.h>
#include <linux/nl80211.h>

static int box_nla_put(struct nlmsghdr *nlh, int maxLength, int type, const void *data, int length) {
	struct nlattr *attr = (struct nlattr *) ((char *) nlh + NLMSG_ALIGN(nlh->nlmsg_len));
	int attrLength = NLA_HDRLEN + length;
	int newLength = NLMSG_ALIGN(nlh->nlmsg_len) + NLA_ALIGN(attrLength);
	if (newLength > maxLength) {
		return -1;
	}
	attr->nla_type = type;
	attr->nla_len = attrLength;
	if (length > 0) {
		memcpy((char *) attr + NLA_HDRLEN, data, length);
	}
	nlh->nlmsg_len = newLength;
	return 0;
}

static void box_nla_parse(struct nlattr **tb, int maxType, struct nlattr *head, int length) {
	memset(tb, 0, sizeof(struct nlattr *) * (maxType + 1));
	while (length >= (int) NLA_HDRLEN) {
		int type = head->nla_type & NLA_TYPE_MASK;
		int attrLength = head->nla_len;
		if (attrLength < (int) NLA_HDRLEN || attrLength > length) {
			return;
		}
		if (type <= maxType) {
			tb[type] = head;
		}
		length -= NLA_ALIGN(attrLength);
		head = (struct nlattr *) ((char *) head + NLA_ALIGN(attrLength));
	}
}

static int box_nl_send(int fd, struct nlmsghdr *nlh) {
	struct sockaddr_nl sa = { .nl_family = AF_NETLINK };
	return sendto(fd, nlh, nlh->nlmsg_len, 0, (struct sockaddr *) &sa, sizeof(sa));
}

static int box_nl80211_resolve(int fd) {
	struct nlmsghdr *nlh = calloc(1, 4096);
	if (nlh == NULL) {
		return -1;
	}
	nlh->nlmsg_len = NLMSG_LENGTH(sizeof(struct genlmsghdr));
	nlh->nlmsg_type = GENL_ID_CTRL;
	nlh->nlmsg_flags = NLM_F_REQUEST;
	struct genlmsghdr *gh = (struct genlmsghdr *) NLMSG_DATA(nlh);
	gh->cmd = CTRL_CMD_GETFAMILY;
	gh->version = 1;
	char familyName[] = "nl80211";
	box_nla_put(nlh, 4096, CTRL_ATTR_FAMILY_NAME, familyName, sizeof(familyName));
	if (box_nl_send(fd, nlh) < 0) {
		free(nlh);
		return -1;
	}
	free(nlh);

	char buffer[4096];
	ssize_t length = recv(fd, buffer, sizeof(buffer), 0);
	if (length < 0) {
		return -1;
	}
	struct nlmsghdr *reply = (struct nlmsghdr *) buffer;
	if (reply->nlmsg_type == NLMSG_ERROR) {
		struct nlmsgerr *err = (struct nlmsgerr *) NLMSG_DATA(reply);
		errno = -err->error;
		return -1;
	}
	struct genlmsghdr *rgh = (struct genlmsghdr *) NLMSG_DATA(reply);
	struct nlattr *tb[CTRL_ATTR_MAX + 1];
	box_nla_parse(tb, CTRL_ATTR_MAX, (struct nlattr *) ((char *) rgh + GENL_HDRLEN),
	              (int) (reply->nlmsg_len - NLMSG_LENGTH(sizeof(struct genlmsghdr))));
	if (tb[CTRL_ATTR_FAMILY_ID] == NULL) {
		return -1;
	}
	return *(int *) ((char *) tb[CTRL_ATTR_FAMILY_ID] + NLA_HDRLEN);
}

static int box_read_ssid(int fd, int familyId, unsigned int ifindex, char *ssid, size_t ssidLength) {
	struct nlmsghdr *nlh = calloc(1, 4096);
	if (nlh == NULL) {
		return -1;
	}
	nlh->nlmsg_len = NLMSG_LENGTH(sizeof(struct genlmsghdr));
	nlh->nlmsg_type = familyId;
	nlh->nlmsg_flags = NLM_F_REQUEST;
	struct genlmsghdr *gh = (struct genlmsghdr *) NLMSG_DATA(nlh);
	gh->cmd = NL80211_CMD_GET_INTERFACE;
	box_nla_put(nlh, 4096, NL80211_ATTR_IFINDEX, &ifindex, sizeof(ifindex));
	if (box_nl_send(fd, nlh) < 0) {
		free(nlh);
		return -1;
	}
	free(nlh);

	char buffer[8192];
	ssize_t length = recv(fd, buffer, sizeof(buffer), 0);
	if (length < 0) {
		return -1;
	}
	struct nlmsghdr *reply = (struct nlmsghdr *) buffer;
	if (reply->nlmsg_type == NLMSG_ERROR) {
		struct nlmsgerr *err = (struct nlmsgerr *) NLMSG_DATA(reply);
		errno = -err->error;
		return -1;
	}
	struct genlmsghdr *rgh = (struct genlmsghdr *) NLMSG_DATA(reply);
	struct nlattr *tb[NL80211_ATTR_MAX + 1];
	box_nla_parse(tb, NL80211_ATTR_MAX, (struct nlattr *) ((char *) rgh + GENL_HDRLEN),
	              (int) (reply->nlmsg_len - NLMSG_LENGTH(sizeof(struct genlmsghdr))));
	if (tb[NL80211_ATTR_SSID] != NULL) {
		int ssidLengthValue = tb[NL80211_ATTR_SSID]->nla_len - NLA_HDRLEN;
		if (ssidLengthValue > 0 && (size_t) ssidLengthValue < ssidLength) {
			memcpy(ssid, (char *) tb[NL80211_ATTR_SSID] + NLA_HDRLEN, ssidLengthValue);
			ssid[ssidLengthValue] = '\0';
		}
	}
	return 0;
}

static int box_read_bssid(int fd, int familyId, unsigned int ifindex, unsigned char *bssid) {
	struct nlmsghdr *nlh = calloc(1, 4096);
	if (nlh == NULL) {
		return -1;
	}
	nlh->nlmsg_len = NLMSG_LENGTH(sizeof(struct genlmsghdr));
	nlh->nlmsg_type = familyId;
	nlh->nlmsg_flags = NLM_F_REQUEST | NLM_F_DUMP;
	struct genlmsghdr *gh = (struct genlmsghdr *) NLMSG_DATA(nlh);
	gh->cmd = NL80211_CMD_GET_SCAN;
	box_nla_put(nlh, 4096, NL80211_ATTR_IFINDEX, &ifindex, sizeof(ifindex));
	if (box_nl_send(fd, nlh) < 0) {
		free(nlh);
		return -1;
	}
	free(nlh);

	char buffer[16384];
	for (;;) {
		ssize_t length = recv(fd, buffer, sizeof(buffer), 0);
		if (length < 0) {
			return -1;
		}
		struct nlmsghdr *message = (struct nlmsghdr *) buffer;
		int remaining = (int) length;
		while (remaining >= (int) sizeof(struct nlmsghdr) &&
		       (int) message->nlmsg_len >= (int) sizeof(struct nlmsghdr) &&
		       (int) message->nlmsg_len <= remaining) {
			if (message->nlmsg_type == NLMSG_DONE) {
				return 0;
			}
			if (message->nlmsg_type == NLMSG_ERROR) {
				struct nlmsgerr *err = (struct nlmsgerr *) NLMSG_DATA(message);
				if (err->error != 0) {
					errno = -err->error;
					return -1;
				}
				return 0;
			}
			struct genlmsghdr *rgh = (struct genlmsghdr *) NLMSG_DATA(message);
			struct nlattr *tb[NL80211_ATTR_MAX + 1];
			box_nla_parse(tb, NL80211_ATTR_MAX, (struct nlattr *) ((char *) rgh + GENL_HDRLEN),
			              (int) (message->nlmsg_len - NLMSG_LENGTH(sizeof(struct genlmsghdr))));
			if (tb[NL80211_ATTR_BSS] != NULL) {
				struct nlattr *bss[NL80211_BSS_MAX + 1];
				box_nla_parse(bss, NL80211_BSS_MAX, (struct nlattr *) ((char *) tb[NL80211_ATTR_BSS] + NLA_HDRLEN),
				              tb[NL80211_ATTR_BSS]->nla_len - NLA_HDRLEN);
				if (bss[NL80211_BSS_STATUS] != NULL && bss[NL80211_BSS_BSSID] != NULL &&
				    *(unsigned int *) ((char *) bss[NL80211_BSS_STATUS] + NLA_HDRLEN) == NL80211_BSS_STATUS_ASSOCIATED) {
					memcpy(bssid, (char *) bss[NL80211_BSS_BSSID] + NLA_HDRLEN, 6);
					return 1;
				}
			}
			int messageLength = (int) NLMSG_ALIGN(message->nlmsg_len);
			remaining -= messageLength;
			message = (struct nlmsghdr *) ((char *) message + messageLength);
		}
	}
}

int box_wifi_state(char *ssid, int ssidLength, unsigned char *bssid) {
	if (ssid == NULL || ssidLength <= 0 || bssid == NULL) {
		return EINVAL;
	}
	ssid[0] = '\0';
	memset(bssid, 0, 6);

	errno = 0;
	unsigned int ifindex = if_nametoindex("wlan0");
	if (ifindex == 0) {
		return errno != 0 ? errno : ENODEV;
	}
	int fd = socket(AF_NETLINK, SOCK_RAW, NETLINK_GENERIC);
	if (fd < 0) {
		return errno != 0 ? errno : EIO;
	}
	errno = 0;
	int familyId = box_nl80211_resolve(fd);
	if (familyId < 0) {
		int error = errno != 0 ? errno : EIO;
		close(fd);
		return error;
	}
	errno = 0;
	if (box_read_ssid(fd, familyId, ifindex, ssid, (size_t) ssidLength) < 0) {
		int error = errno != 0 ? errno : EIO;
		close(fd);
		return error;
	}
	if (ssid[0] != '\0') {
		box_read_bssid(fd, familyId, ifindex, bssid);
	}
	close(fd);
	return 0;
}
