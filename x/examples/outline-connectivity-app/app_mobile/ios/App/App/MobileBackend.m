//
//  Library.m
//  App
//
//  Created by Daniel LaCosse on 8/7/23.
//

#import <Foundation/Foundation.h>
#import <Capacitor/Capacitor.h>

CAP_PLUGIN(BackendPlugin, "MobileBackend", CAP_PLUGIN_METHOD(Invoke, CAPPluginReturnPromise);)
