// Copyright 2023 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Questions:
// - using HttpClient in dart:io, or Client in dart:http?
// - How exactly is it being used? Need to see calls in code.

// Findings for HttpClient:
// - Can select different proxies based on target address.
// - Can't disable TLS
// - Can't change target address and select a proxy at the same time. Either one or the other.
// - We may be able to do more with dart:http BaseClient.send.

// ignore_for_file: unnecessary_this

import 'dart:convert';

import 'package:args/args.dart';
import 'dart:io';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';

HttpClient makeClientWithFindProxy(String? proxy) {
  var client = HttpClient();
  client.connectionTimeout = Duration(seconds: 10);
  if (proxy == null) {
    return client;
  }

  // Configure proxy
  client.findProxy = (uri) {
    return "PROXY $proxy";
  };
  return client;
}

Future<ConnectionTask<Socket>> connectHttp(
    String proxyHost, int proxyPort, String targetHost, int targetPort) async {
  var socketTask = Socket.startConnect(proxyHost, proxyPort);
  var client = HttpClient()
    ..connectionFactory = (url, proxyHost, proxyPort) => socketTask;

  var request =
      await client.openUrl("CONNECT", Uri(host: proxyHost, port: proxyPort));
  // if (uri.scheme == "https") {
  //   request.headers.add("Transport", "split:2");
  // }
  var response = await request.close();
  response.detachSocket();
  return socketTask;
}

//   return socketTask;

//   proxySocket.write('CONNECT $targetHost:$targetPort HTTP/1.1\r\n');
//   proxySocket.write('Host: $targetHost:$targetPort\r\n');
//   proxySocket.write('Connection: keep-alive\r\n');
//   proxySocket.write('\r\n');
//   await proxySocket.flush();

//   // Read the response from the socket
//   Completer<Socket> completer = Completer<Socket>();
//   String response = '';
//   while(proxySocket.take  moveNext())
//   proxySocket.listen(
//     (data) {
//       response += utf8.decode(data);
//       // Look for the end of the response headers (indicated by a double CRLF)
//       if (response.indexOf('\r\n\r\n') != -1) {
//         // The response headers are complete
//         final headers = response.split('\r\n');
//         // Typically, you would parse the headers and check the response status code.
//         // Here, for simplicity, we just print the response and complete the completer.
//         print(response); // For debug: see what the response is
//         // Check for a successful response (HTTP 200)
//         if (headers.any((h) => h.startsWith('HTTP/1.1 200'))) {
//           completer.complete(socket);
//         } else {
//           socket.destroy();
//           completer.completeError(
//             HttpException('Proxy failed to establish a connection'),
//           );
//         }
//       }
//     },
//     onError: (error) {
//       socket.destroy();
//       completer.completeError(error);
//     },
//     onDone: () {
//       if (!completer.isCompleted) {
//         completer.completeError(
//           HttpException('Socket has been closed before the response was complete'),
//         );
//       }
//     },
//     cancelOnError: true,
//   );

//   return completer.future;
// }

HttpClient makeClientWithConnectionFactory(String proxyHost, int proxyPort) {
  var client = HttpClient();
  // Configure proxy
  client.connectionFactory = (uri, nullHost, nullPort) async {
    assert(nullHost == null);
    assert(nullPort == null);
    // Future<ConnectionTask<Socket>> futureSocket;
    // switch (uri.host) {
    //   case "www.rferl.org":
    //     futureSocket = Socket.startConnect("e4887.dscb.akamaiedge.net", uri.port);
    //     break;
    //   default:
    //     futureSocket = Socket.startConnect(uri.host, uri.port);
    //     break;
    // }
    // if (uri.scheme == "https") {
    //   SecureSocket.s
    // }

    var connectHost = uri.host;
    if (uri.host == "www.rferl.org") {
      connectHost = "e4887.dscb.akamaiedge.net";
    }
    if (uri.scheme == "https") {
      return SecureSocket.startConnect(connectHost, uri.port);
    }
    return Socket.startConnect(connectHost, uri.port);
    // This results in Bad state: Stream has already been listened to.
    // return connectHttp(proxyHost, proxyPort, uri.host, uri.port);
    // var request = await proxyClient.openUrl("CONNECT", Uri(host: proxyUri.host, port: proxyUri.port));
    // if (uri.scheme == "https") {
    //   request.headers.add("Transport", "split:2");
    // }
    // var response = await request.close();
    // var futureSocket = response.detachSocket();
    // return ConnectionTask<Socket>._(
    //   socket: futureSocket,
    //   cancel: () async {
    //     (await futureSocket).close();
    //   },
    // );
    // var socket = await Socket.connect(proxyUri.host, proxyUri.port);
    // socket.write("CONNECT ${uri.host}:${uri.port} HTTP/1.1\r\n");
    // socket.write("Host: ${uri.host}:${uri.port}\r\n");
    // socket.write('\r\n');
    // await socket.flush();
  };
  return client;
}

http.Client makeRemapClient(String proxy) {
  var ioClient = HttpClient()
    ..findProxy = (uri) {
      if (uri.port == 443) {
        print("Proxy for $uri: $proxy");
        return "PROXY $proxy";
      }
      print("Proxy for $uri: DIRECT");
      return "DIRECT";
    };
  return RemapClient(IOClient(ioClient));
}

class RemapClient extends http.BaseClient {
  final http.Client _client;

  RemapClient(http.Client client) : _client = client;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) {
    print("Called send with $request");

    // Create a new request with the modified URL
    var url = request.url;
    if (url.scheme == "https") {
      var port = url.hasPort ? url.port : 443;
      url = url.replace(scheme: "http", port: port);
    }
    if (url.host == 'www.rferl.org') {
      url = url.replace(host: "e4887.dscb.akamaiedge.net");
    }
    print("New url is $url");
    var newRequest = http.Request(request.method, url)
      ..headers.addAll(request.headers)
      ..followRedirects = false // request.followRedirects
      ..maxRedirects = request.maxRedirects
      ..persistentConnection = request.persistentConnection;

    if (request is http.Request) {
      newRequest.body = request.body;
      newRequest.encoding = request.encoding;
    }
    print("New request is $newRequest");

    // Call the base class's send method
    return this._client.send(newRequest);
  }

  @override
  void close() {
    this._client.close();
  }
}

Future<bool> testIoClient(HttpClient client, Uri url) async {
  try {
    var request = await client.headUrl(url);
    await request.close();
    return true;
  } catch (e) {
    print(e);
    return false;
  }
}

Future<bool> testHttpClient(http.Client client, Uri url) async {
  try {
    var response = await client.get(url);
    print("Status: ${response.statusCode}");
    print("Headers: ${response.headers}");
    print("Response: ${utf8.decode(response.bodyBytes)}");
    return true;
  } catch (e) {
    print(e);
    return false;
  }
}

void main(List<String> arguments) async {
  final parser = ArgParser()..addOption('proxy', abbr: 'p', mandatory: false);

  ArgResults argResults;

  try {
    argResults = parser.parse(arguments);
  } on ArgParserException catch (e) {
    print('Error parsing arguments: $e');
    print('Usage: dart script.dart [--proxy <proxy>] <url>');
    exit(1);
  }

  // Validate and extract URL argument
  final List<String> rest = argResults.rest;
  if (rest.isEmpty) {
    print('No URL provided.');
    print('Usage: dart script.dart --proxy <proxy> <url>');
    exit(1);
  }
  final String urlString = rest.last;
  final Uri? url = Uri.tryParse(urlString);
  if (url == null) {
    print('Invalid URL: $urlString');
    exit(1);
  }

  // Extract proxy argument
  final String? proxyAddress = argResults['proxy'];
  if (proxyAddress == null) {
    return;
  }

  // HttpClient client = HttpClient();
  // bool isClientOk = await testIoClient(client, url);
  // print("Direct access: ${isClientOk ? "OK" : "FAILED"}");
  // client.close(force: true);
  {
    final client = makeRemapClient(proxyAddress);
    var isClientOk = await testHttpClient(client, url);
    print("Direct access (RemapClient): ${isClientOk ? "OK" : "FAILED"}");
    client.close();
  }

  // try {
  //   var client = http.Client();
  //   isClientOk = await testHttpClient(client, url);
  //   print("Direct access (http.Client): ${isClientOk ? "OK" : "FAILED"}");
  // } finally {
  //   client.close();
  // }

  // client = makeClientWithFindProxy(proxyAddress);
  // isClientOk = await testIoClient(client, url);
  // print("Proxy access (findProxy): ${isClientOk ? "OK" : "FAILED"}");
  // client.close(force: true);

  // final proxyUri = Uri.parse("http://$proxyAddress");
  // client = makeClientWithConnectionFactory(proxyUri.host, proxyUri.port);
  // isClientOk = await testIoClient(client, url);
  // print("Proxy access (HTTP CONNECT): ${isClientOk ? "OK" : "FAILED"}");

  // if (isClientOk) {
  //   try {
  //     var request = await client.getUrl(url);
  //     var response = await request.close();

  //     if (response.statusCode == 200) {
  //       print('Page fetched successfully:');
  //       // Read the response body
  //       await for (var contents in response.transform(utf8.decoder)) {
  //         print(contents);
  //       }
  //     } else {
  //       print('Failed to fetch page: ${response.statusCode}');
  //     }
  //   } catch (e) {
  //     print('Error fetching page: $e');
  //   } finally {
  //     client.close();
  //   }
  // }
  // client.close();
}
