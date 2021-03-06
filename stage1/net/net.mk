LOCAL_ACI_CONFDIR := $(ACIROOTFSDIR)/etc/rkt/net.d
LOCAL_CONFFILES := $(wildcard $(MK_SRCDIR)/conf/*.conf)

$(call setup-stamp-file,LOCAL_STAMP)

define LOCAL_SRC_TO_ACI_CONFFILE
$(LOCAL_ACI_CONFDIR)/$(notdir $1)
endef

LOCAL_ACI_CONFFILES :=
LOCAL_INSTALL_TRIPLETS :=
$(foreach c,$(LOCAL_CONFFILES), \
        $(eval _NET_MK_ACI_CONFFILE_ := $(call LOCAL_SRC_TO_ACI_CONFFILE,$c)) \
        $(eval LOCAL_ACI_CONFFILES += $(_NET_MK_ACI_CONFFILE_)) \
        $(eval LOCAL_INSTALL_TRIPLETS += $c:$(_NET_MK_ACI_CONFFILE_):-))

$(LOCAL_STAMP): $(LOCAL_ACI_CONFFILES)
	touch "$@"

STAGE1_INSTALL_FILES += $(LOCAL_INSTALL_TRIPLETS)
STAGE1_INSTALL_DIRS += $(LOCAL_ACI_CONFDIR):0755
STAGE1_STAMPS += $(LOCAL_STAMP)

$(call undefine-namespaces,LOCAL _NET_MK)
