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


import 'dart:js_interop';

import 'package:args/args.dart';
import 'dart:io';
import 'dart:convert';

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

Future<ConnectionTask<Socket>> connectHttp(String proxyHost, int proxyPort, String targetHost, int targetPort) async {
  var socketTask = Socket.startConnect(proxyHost, proxyPort);
  var client = HttpClient()
    ..connectionFactory =(url, proxyHost, proxyPort) => socketTask;
  
  var request = await client.openUrl("CONNECT", Uri(host: proxyHost, port: proxyPort));
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

Future<bool> testClient(HttpClient client, Uri url) async {
  try {
    var request = await client.headUrl(url);
    await request.close();
    return true;
  } catch (e) {
    print(e);
    return false;
  }
}

void main(List<String> arguments) async {
  final parser = ArgParser()
    ..addOption('proxy', abbr: 'p', mandatory: false);

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

  HttpClient client = HttpClient();
  bool isClientOk = await testClient(client, url);
  print("Direct access: ${isClientOk ? "OK" : "FAILED"}");
  client.close(force: true);

  // Extract proxy argument
  final String? proxyAddress = argResults['proxy'];
  if (proxyAddress == null) {
    return;
  }

  client = makeClientWithFindProxy(proxyAddress);
  isClientOk = await testClient(client, url);
  print("Proxy access (findProxy): ${isClientOk ? "OK" : "FAILED"}");
  client.close(force: true);

  final proxyUri = Uri.parse("http://$proxyAddress");
  client = makeClientWithConnectionFactory(proxyUri.host, proxyUri.port);
  isClientOk = await testClient(client, url);
  print("Proxy access (HTTP CONNECT): ${isClientOk ? "OK" : "FAILED"}");

  if (isClientOk) {
    try {
      var request = await client.getUrl(url);
      var response = await request.close();
      
      if (response.statusCode == 200) {
        print('Page fetched successfully:');
        // Read the response body
        await for (var contents in response.transform(utf8.decoder)) {
          print(contents);
        }
      } else {
        print('Failed to fetch page: ${response.statusCode}');
      }
    } catch (e) {
      print('Error fetching page: $e');
    } finally {
      client.close();
    }
  }
  client.close();
}