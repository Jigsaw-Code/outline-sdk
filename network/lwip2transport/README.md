# LwIP Packet Traversal Sequence

This document tracks the traversal of packets through the `lwip2transport` LwIP network stack implementation, bridging the local host's TUN device and a remote proxy socket.

## Architecture & State Management

Before looking at the packet flows, it is important to understand how associations (connections) are managed between the TUN interface and the proxy.

* **Global Map (`udpConns`)**: The association state is managed by a global map of active connections, keyed by the source address.
* **`UDPConn`**: Represents a single association. It stores the necessary state, primarily the destination local IP, determining where return packets should be sent.
* **Association Lifecycle**: 
  - **Created**: When `udpRecvFn` receives a packet from a new source address.
  - **Closed**: Explicitly closed by the application (e.g., in `udpHandler.closeSession()`) or when the entire LwIP stack is shut down. The LwIP stack does *not* automatically time out or close these UDP associations. If a `UDPConn` is closed, the next packet from that source will simply create a new one.

---

## 1. Outbound Flow (Send Sequence): Local TUN Device to Proxy

This flow handles packets originating from the local device and heading out to the internet via the proxy.

```mermaid
sequenceDiagram
    autonumber
    participant App as App Loop
    participant TUN as Local TUN Device
    participant Dev as lwIPDevice
    participant LWIP as LwIP Stack (C)
    participant URF as udpRecvFn (Go Export)
    participant Handler as UDPConnHandler
    participant Proxy as Proxy Sender
    participant Remote as Remote Server

    loop While App Running
        App->>TUN: Read outgoing IP packet
        TUN-->>App: Return IP packet
        App->>Dev: Write(ipPacket)
        Dev->>LWIP: LWIPStack.Write(ipPacket) -> input()
        Note over LWIP: Processes IP headers,<br/>extracts UDP payload.
        LWIP->>URF: udpRecvFn(packet)
        Note over URF: Looks up or creates UDPConn<br/>in global udpConns map.
        URF->>Handler: ReceiveTo(udpConn, payload, destAddr)
        Note over Handler: If new session:<br/>Calls proxy.NewSession(respWriter)<br/>Starts Proxy read loop.
        Handler->>Proxy: WriteTo(payload, destAddr)
        Proxy->>Remote: Send payload via Proxy socket
    end
```

### Step-by-Step Walkthrough

#### 1. Read from Local TUN
The VPN application runs a continuous loop reading outbound IP packets from the host OS TUN device. (On Apple, the client calls [NEPacketTunnelFlow.readPackets](https://developer.apple.com/documentation/networkextension/nepackettunnelflow/readpackets(completionhandler:)) and gets a callback from the OS instead of reading directly from the TUN device).

#### 2. Write to `lwIPDevice`
The app writes the IP packet to the `lwIPDevice` (`lwIPDevice.Write`).

#### 3. Ingested by LwIP
`lwIPDevice.Write` pushes the raw packet into the LwIP stack (`LWIPStack.Write` which calls LwIP's `input()` function). 

#### 4. LwIP Processing & `udpRecvFn`
LwIP processes the IP headers. For UDP packets, it invokes the Go-exported callback `udpRecvFn` (`go-tun2socks.udp_callback_export`). 
* `udpRecvFn` handles the association logic. It queries the global `udpConns` map using the packet's source address. 
* If no association exists, it creates a new `UDPConn` (via `newUDPConn`), which registers the local and remote addresses.

#### 5. Hand-off to `UDPConnHandler`
`udpRecvFn` then calls the `ReceiveTo` method on the globally registered `UDPConnHandler`, passing the `UDPConn`, the payload, and the destination address.

#### 6. Session Creation & Proxying
`UDPConnHandler.ReceiveTo` checks if a proxy session exists for this `UDPConn`. 
* If new, it creates a `udpConnResponseWriter` and calls `proxy.NewSession()`, which typically spins up a background goroutine to listen for returning proxy traffic.
* It then delegates the payload to the proxy sender (`reqSender.WriteTo`), which writes it to the remote server via the proxy socket.

---

## 2. Inbound Flow (Receive Sequence): Proxy to Local TUN Device

This flow handles return traffic from the remote server, through the proxy, back to the local TUN device.

```mermaid
sequenceDiagram
    autonumber
    participant App as App Loop
    participant Dev as lwIPDevice
    participant Remote as Remote Server
    participant Proxy as Proxy Listener Loop
    participant RW as udpConnResponseWriter
    participant LWIP as LwIP Stack (C)
    participant TUN as Local TUN Device

    App->>Dev: Read(buffer)
    Note over Dev: Blocks waiting for packet on<br/>d.rdBuf channel.

    loop Proxy Listener
        Remote->>Proxy: Receive response payload
        Proxy->>RW: WriteFrom(payload, sourceAddr)
        Note over RW: Resolves net.Addr to *net.UDPAddr
        RW->>LWIP: udpConn.WriteFrom(payload, srcAddr)
        Note over LWIP: udpConn calls C.udp_sendto().<br/>LwIP encapsulates UDP payload<br/>into an IP packet using<br/>associated local IP.
        LWIP->>Dev: forwardOutgoingIPPacket(ipPacket)
        Note over Dev: Hands the IP packet to the consumer<br/>over the unbuffered d.rdBuf channel<br/>and waits for d.rdN.
    end

    Dev-->>App: Unblocks, returns IP packet
    App->>TUN: Write IP packet to local TUN
```

### Step-by-Step Walkthrough

#### 1. App Loop Waits for Packets
The application has a loop that continuously calls `lwIPDevice.Read()`. This call blocks, waiting for packets to arrive on the `d.rdBuf` channel.

#### 2. Proxy Receives Response
The background loop (spawned during the Outbound Flow) continuously reads from the proxy socket. It receives a UDP response payload from the remote server.

#### 3. Write to `udpConnResponseWriter`
The proxy listener passes the payload to `WriteFrom(payload, sourceAddr)` on the `udpConnResponseWriter` associated with this session.

#### 4. Hand-off to `UDPConn` & LwIP
The writer resolves the source address and calls `WriteFrom` on the underlying `UDPConn`.
* `UDPConn` contains the receiving packet logic. It calls the C function `C.udp_sendto()`, passing the payload along with the local and remote addresses stored in the association.
* The LwIP stack encapsulates the raw UDP payload into a full IP packet. The destination IP is determined by the local IP state held in the `UDPConn` association.

#### 5. LwIP Outputs IP Packet
Once LwIP constructs the IP packet, it calls its registered output callback: `forwardOutgoingIPPacket` (registered with `lwip.RegisterOutputFn`).

#### 6. Sent to `lwIPDevice` Read Channel
`forwardOutgoingIPPacket` hands the resulting IP packet to the consumer over the `lwIPDevice`'s unbuffered `d.rdBuf` channel and waits for the byte count on `d.rdN`. The handoff is a rendezvous because the packet aliases the C pbuf payload, which LwIP frees as soon as the output callback returns — the consumer must copy it out before signalling `d.rdN`.

#### 7. Read by App & Written to TUN
The blocked `lwIPDevice.Read()` call (from Step 1) unblocks, copies the packet from `d.rdBuf`, and returns it to the application loop. The application then writes this complete IP packet back to the host OS local TUN interface.

#### Alternative: `WriteTo` and the async writer
Instead of looping on `Read`, an application can hand the device a destination writer once via `lwIPDevice.WriteTo(w)` (this is what the Outline clients do with their TUN file). The critical constraint on this path is that **LwIP calls `forwardOutgoingIPPacket` while holding its global stack lock**: any blocking work there — notably a `write(2)` to the TUN device — stalls every TCP and UDP flow in the stack for the duration of the syscall.

So `WriteTo` never writes to `w` inline. It wraps `w` in an `asyncWriter` (`async_writer.go`), which:

1. copies the packet into a pooled buffer (`lwIPDevice.pktPool`) and queues it (`writeQueueDepth` = 256 slots), returning immediately — this is what releases the pbuf and lets `WriteTo` signal `d.rdN`;
2. performs the actual `w.Write` calls from a background goroutine, recycling each buffer afterwards.

Consequences worth knowing:

* **Backpressure** still applies: once the queue is full, `asyncWriter.Write` blocks, which blocks `WriteTo`, which blocks the LwIP callback. That happens only when the destination is genuinely slower than the stack is producing.
* **Byte counts** returned by `WriteTo` are bytes *queued*, not bytes the destination has accepted.
* **Write errors** surface one or more packets after the packet that caused them. `WriteTo` returns as soon as it observes the error, and packets still queued at that point are dropped.
* Closing the device (or returning from `WriteTo`) stops the background goroutine; it waits for the in-flight write but drops the rest of the queue rather than letting a stuck destination block shutdown.
