#ifndef PLUGIN_FACTORY_H
#define PLUGIN_FACTORY_H

#include "bmw_list.h"

struct PluginAppInfo {
    unsigned abiVersion;
    const char* name;
    const char* vendor;
    unsigned interfaceVersion;
    bmw::PluginApp* (*create)(bmw::IDispatcher&, const char*, const bmw::Bundle&, const bmw::Bundle&);
    void (*destroy)(bmw::PluginApp*);
};

#if defined(__arm__)
typedef char plugin_descriptor_size[(sizeof(PluginAppInfo)==24) ? 1 : -1];
typedef char plugin_base_size[(sizeof(bmw::PluginApp)==352) ? 1 : -1];
typedef char plugin_string_size[(sizeof(bmw::BasicString<char>)==4) ? 1 : -1];
#endif

template <class App>
static bmw::PluginApp* create_app(bmw::IDispatcher& dispatcher, const char* name,
                                  const bmw::Bundle& config, const bmw::Bundle& resources) {
    bmw::RHMIInterfaceVersion version = {1};
    return new App(dispatcher, bmw::BasicString<char>(name), 1u, 0u, version, config, resources);
}

static inline void destroy_app(bmw::PluginApp* app) { delete app; }

#endif
