#!/bin/bash

npx rimraf \
  capacitor.config.json \
  node_modules/ \
  ios/capacitor-cordova-ios-plugins/ \
  ios/DerivedData/ \
  ios/App/Pods/ \
  ios/App/App/Config.swift \
  ios/App/App/Info.plist \
  ios/App/App/public \
  ios/App/App/capacitor.config.json \
  ios/App/App/config.xml \
  android/.gradle/ \
  android/.idea/ \
  android/capacitor-cordova-android-plugins/ \
  android/local.properties \
  android/app/build/ \
  android/app/src/main/java/org/getoutline/pwa/Config.kt