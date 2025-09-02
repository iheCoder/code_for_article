# WebRTC 探秘：构建你自己的实时视频应用

## 导言：揭秘 Web 端的实时通信



### 定义 WebRTC：从插件到原生浏览器 API 的范式转变

WebRTC（Web Real-Time Communication，Web 实时通信）是一项免费的开源项目，也是由万维网联盟（W3C）和互联网工程任务组（IETF）共同制定的一套标准。其核心功能是通过 JavaScript 应用程序编程接口（API）为 Web 浏览器和移动应用程序提供实时通信（RTC）能力。

WebRTC 的核心价值主张在于，它彻底消除了在 Web 端实现实时通信长期以来对专有插件、浏览器扩展或原生应用程序下载的依赖。在 WebRTC 出现之前，用户通常需要安装 Flash、Silverlight 或特定的桌面应用才能进行视频聊天或语音通话。WebRTC 将这些功能原生集成到浏览器中，极大地降低了开发门槛，使实时互动功能的普及成为可能。目前，该技术已获得所有主流浏览器（包括 Chrome、Firefox、Safari 和 Edge）在桌面和移动平台的广泛支持，成为一项无处不在的行业标准 。



### 解决的根本问题：实现安全、低延迟的点对点媒体与数据流

WebRTC 旨在解决的核心挑战是，在充满不可预测性和网络限制的公共互联网拓扑中，如何在两个终端（即“对等端”或“Peer”）之间建立直接、低延迟的通信信道。这些信道专为音频、视频以及任意数据的传输而设计，构成了现代通信应用的基石。在理想的网络条件下，WebRTC 的延迟通常在 100-300 毫秒之间，完全满足实时互动的要求。

值得强调的是，安全性在 WebRTC 的设计中并非事后添加的补充，而是一项强制性的内置特性。所有 WebRTC 通信都必须进行加密传输（使用 DTLS 和 SRTP 协议）以确保链路上的机密性与完整性。若要实现真正的端到端加密（即服务器也无法看到明文，在使用 SFU 等中间节点时仍保持加密），需要结合 Insertable Streams/SFrame 在应用层实现额外的端到端加密。



### 关键应用与行业影响：从视频会议到去中心化文件共享与物联网

WebRTC 的诞生催生了众多创新应用，其影响力已渗透到各行各业：

- **远程医疗（Telehealth）：** 实现医生与患者之间的高清视频通话，并能安全地共享健康记录等敏感数据。
- **在线教育与协作（E-learning & Collaboration）：** 构建虚拟教室、实时白板和协同编辑工具，打破地域限制。
- **客户支持与云通信（Customer Support & Cloud Phones）：** 通过浏览器内嵌的 VoIP“云电话”，用户无需离开网页即可直接与客服代表通话。
- **去中心化内容分发（Decentralized Content Delivery）：** 支持浏览器间的点对点文件共享（如 WebTorrent），以及利用用户带宽分发内容的 CDN 网络。
- **物联网与智能家居（IoT & Smart Home）：** 允许移动应用与智能设备（如视频门铃、网络摄像头）进行直接通信，实现实时警报和视频流传输。

一个普遍存在的误解是，WebRTC 是一种“无服务器”（Serverless）的技术。虽然其最终目标是建立一个媒体数据直接在对等端之间传输的点对点（Peer-to-Peer, P2P）连接，但实现这一目标的全过程——从发现对等端、协商会话参数到最终建立连接——都严重依赖于一系列服务器的协同工作，包括信令服务器、STUN 服务器和 TURN 服务器。因此，WebRTC 并非消除了服务器，而是重新定义了它们在通信架构中的角色：从传统的媒体中继者转变为连接的“促成者”和“协调者”。WebRTC 规范明确指出，其本身“未包含信令相关的规定”。这意味着开发者必须自行实现或部署一个服务器端组件来管理对等端的发现和会话协商过程。此外，为了解决普遍存在的网络地址转换（NAT）问题，WebRTC 必须借助外部辅助服务。STUN 服务器用于帮助客户端发现其公网 IP 地址 ，而 TURN 服务器则在直接连接失败时充当备用的中继服务器。因此，一个功能完备的 WebRTC 应用并非无服务器架构，而是一个复杂的分布式系统。在这个系统中，媒体平面（Media Plane）在理想情况下是去中心化的（P2P），但控制平面（Control Plane）和建立连接的辅助平面则依赖于开发者管理的信令服务器和可公开访问的 STUN/TURN 服务器。理解这一点对于准确评估构建 WebRTC 服务的架构复杂性和运营成本至关重要。



## 第 1 节：WebRTC 的架构支柱

本节将深入探讨构成 WebRTC 技术的各个核心组件，不仅解释它们“是什么”，更阐明它们“为何”如此设计。



### 1.1 信令平面：未被规定的先决条件

WebRTC 标准有意地没有规定具体的信令协议，这是一个深思熟虑的设计决策。其目的是为了赋予开发者最大的灵活性，允许他们使用任何适合自身业务场景的协议（如 SIP、XMPP 或基于 WebSocket 的自定义协议）将 WebRTC 集成到现有系统中。

尽管协议未定，但信令服务器在 WebRTC 连接建立过程中扮演着不可或缺的核心角色。其主要职责可归纳为三点：

1. **对等端发现与会话管理（Peer Discovery & Session Management）：** 帮助希望通信的用户找到彼此，并管理通信会话的整个生命周期，包括初始化、修改和终止。
2. **媒体协商（Media Negotiation）：** 负责在对等端之间中继包含 SDP（会话描述协议）的 `offer`（提议）和 `answer`（应答）消息。这些消息描述了各端的媒体能力，如支持的音视频编解码器。
3. **网络路径协商（Network Path Negotiation）：** 负责中继 ICE 候选者（ICE Candidates）。这些候选者描述了每个对等端可能用于建立连接的网络地址。

在众多信令传输技术中，WebSocket 因其持久、双向、低延迟的特性而成为最受欢迎的选择之一。相比之下，传统的 HTTP 长轮询等方式效率较低。为保证通信安全，信令通道本身也应使用加密协议，如 HTTPS 或 WSS 4。



### 1.2 会话描述协议 (SDP)：协商媒体合同

SDP（Session Description Protocol）是一种基于文本的格式，其作用是描述多媒体会话的参数，而非一种传输协议。它的设计初衷是传递足够的信息，使接收方能够成功加入并参与会话。

一个典型的 SDP 消息包含了建立媒体流所需的全部元数据，例如：

- **媒体类型：** 声明会话包含音频（`audio`）还是视频（`video`）。
- **编解码器（Codecs）：** 列出发送方支持的音视频编解码器，如 Opus（音频）、VP8、H.264（视频）等。
- **加密算法：** 指定用于保护媒体流安全的加密套件（如 DTLS-SRTP）。
- **网络信息：** （在传统 ICE 模式下）包含用于接收媒体的 IP 地址和端口信息。

WebRTC 采用了一种在 RFC 3264 中标准化的“提议/应答”（Offer/Answer）模型来进行媒体协商。其流程如下：一方（提议方，Offerer）生成一个包含其期望会话配置的 SDP `offer`；另一方（应答方，Answerer）在收到 `offer` 后，生成一个 SDP `answer` 作为回应，其中包含了它所能支持的兼容配置。通过这一来一回的交换，双方就本次通信的技术细节（如使用哪种编解码器）达成了一致。

此外，现代 WebRTC 实践通常配合 BUNDLE 与 RTCP-mux，将多路媒体复用在单一传输之上，m= 行端口常为占位的 9，具体可用地址与端口主要通过 ICE 候选者在信令过程中逐步交换（Trickle ICE）。因此，SDP 本体更多承载“媒体合同”（编解码能力、方向性、复用策略），而非固定的网络寻址信息。



### 1.3 穿越网络屏障：NAT 穿透框架

#### NAT 问题

网络地址转换（NAT）是 P2P 通信面临的根本性障碍。绝大多数设备都位于 NAT 路由器之后，拥有的是私有、不可路由的 IP 地址。这使得它们对于公共互联网而言是“不可见”的，外部设备无法直接向其发起连接请求，从而阻碍了 P2P 通信的建立 7。



#### STUN (Session Traversal Utilities for NAT)

- **功能：** STUN 服务器是一种轻量级网络工具，其唯一目的是帮助位于 NAT 后面的客户端发现自己的公网 IP 地址和端口号，这个地址也被称为“服务器反射地址”（Server Reflexive Address）。
- **机制：** 客户端向 STUN 服务器发送一个“绑定请求”（Binding Request）。STUN 服务器收到请求后，会检查该数据包的源 IP 地址和端口，然后将这个地址信息作为响应返回给客户端。
- **局限性：** STUN 对许多类型的 NAT（如完全锥形 NAT、受限锥形 NAT）都非常有效。然而，它对于一种更严格的 NAT 类型——“对称 NAT”（Symmetric NAT）——则无能为力。在对称 NAT 环境下，路由器会为每个不同的目标地址和端口组合都分配一个新的、唯一的公网端口，这使得通过 STUN 发现的地址对其他对等端无效。



#### TURN (Traversal Using Relays around NAT)

- **功能：** TURN 服务器是媒体中继服务器，是当直接 P2P 连接（即使有 STUN 的帮助）失败时的最终备用方案。
- **机制：** 当 P2P 连接尝试失败时，通信双方会各自连接到 TURN 服务器。客户端会在 TURN 服务器上被分配一个“中继传输地址”（Relayed Transport Address），并将其告知通信对端。此后，所有媒体数据包都将通过 TURN 服务器进行转发。
- **权衡：** TURN 能够保证在最复杂的网络环境下也能建立连接，但这是有代价的。首先，媒体数据需要经过服务器中转，增加了额外的网络跳数，从而提高了延迟。其次，由于需要处理所有媒体流，TURN 服务器会消耗大量的带宽和计算资源，其运营和扩展成本远高于 STUN 服务器。

STUN 和 TURN 之间的选择不仅仅是 ICE 算法在技术层面的决策，它对任何 WebRTC 服务的提供商都具有深远的经济和架构影响。STUN 服务器是无状态、轻量级的，运营成本较低；而 TURN 服务器需要为会话中继媒体数据，带来显著的带宽与部署成本，并需要在全球多点就近部署以降低时延。

在实际互联网环境中，多数会话可通过直连（含 STUN 协助）成功，但仍有一部分会话在对称 NAT、企业防火墙或严格代理环境下需要 TURN 兜底。该比例会随用户地区、网络形态与业务人群而变化，无法一概而论。工程上应按“最坏情况可用”的原则，规划并容量评估 TURN，以保证在直连失败时仍能建立连接。这直接引出了一系列关键的架构决策：自建（如 `coturn`）以获得可控性与成本可预期，或采用托管云服务以快速获得全球可用性与弹性。



### 1.4 交互式连接建立 (ICE)：总控框架

ICE（Interactive Connectivity Establishment），在 RFC 8445 中定义，是一个综合性的框架。它系统性地协调 STUN 和 TURN 的使用，以期在两个对等端之间找到最理想的通信路径。



#### ICE 候选者 (ICE Candidate)

ICE 候选者是一个数据对象，代表了一个对等端可能被访问到的网络地址。在 ICE 流程中，会收集以下几种主要类型的候选者：

1. **主机候选者 (Host)：** 设备的本地私有 IP 地址。如果通信双方在同一个局域网内，这种方式最有效。
2. **服务器反射候选者 (Server Reflexive, srflx)：** 通过 STUN 服务器发现的公网 IP 地址和端口。
3. **中继候选者 (Relayed, relay)：** TURN 服务器为客户端分配的 IP 地址和端口，用于中继媒体数据。



#### Trickle ICE 优化

最初的 ICE 规范流程（有时被称为“Vanilla ICE”）是串行的，效率较低：首先，需要等待收集完 *所有* 类型的候选者；然后，将它们一次性打包在 SDP 消息中交换；最后，才开始进行连通性检查。这个过程会给通话建立带来显著的延迟。

为了解决这个问题，Trickle ICE（在 RFC 8838 中定义）被引入，它极大地优化了这一过程。Trickle ICE 允许候选者在被发现后，立即通过信令通道“涓滴”（trickle）式地发送给对端。这使得候选者的收集、交换和连通性检查这三个阶段可以并行进行，从而大幅缩短了建立连接所需的时间。现代的 WebRTC 实现默认都使用 Trickle ICE。

| 术语/组件  | 定义                                                         |
| ---------- | ------------------------------------------------------------ |
| **WebRTC** | 一套允许浏览器和移动应用进行点对点实时音视频和数据通信的开放标准和 API。 |
| **API**    | 应用程序编程接口，是开发者用来与 WebRTC 功能交互的一组 JavaScript 方法和事件。 |
| **P2P**    | 点对点（Peer-to-Peer），指数据直接在两个终端设备之间传输，无需经过中心服务器中转。 |
| **SDP**    | 会话描述协议（Session Description Protocol），一种用于描述多媒体会话参数的文本格式。 |
| **NAT**    | 网络地址转换（Network Address Translation），一种将私有网络地址映射到公共网络地址的技术。 |
| **STUN**   | NAT 会话穿透功能（Session Traversal Utilities for NAT），一种帮助客户端发现其公网 IP 地址的协议。 |
| **TURN**   | 使用中继穿透 NAT（Traversal Using Relays around NAT），一种在无法直接连接时充当媒体中继的协议。 |
| **ICE**    | 交互式连接建立（Interactive Connectivity Establishment），一个综合利用 STUN 和 TURN 寻找最佳 P2P 连接路径的框架。 |
| **DTLS**   | 数据报传输层安全（Datagram Transport Layer Security），用于加密 WebRTC 数据通道和协商 SRTP 密钥的协议。 |
| **SRTP**   | 安全实时传输协议（Secure Real-time Transport Protocol），用于加密和验证 WebRTC 音视频媒体流的协议。 |



## 第 2 节：WebRTC 连接剖析：分步详解

本节将把第 1 节中介绍的各个组件串联起来，以一个清晰的时间顺序，完整地描述一个 WebRTC 连接的建立过程。我们假设有两个对等端：“Alice”（呼叫方/发起方）和“Bob”（被叫方/接收方）。



### 2.1 阶段一：发起与媒体捕获

通信过程由 Alice 的浏览器发起。首先，它会调用 `navigator.mediaDevices.getUserMedia()` API，请求访问用户的摄像头和麦克风。在获得用户授权后，该 API 会返回一个 `MediaStream` 对象。这个对象是媒体数据的源头，其中包含了代表音频和视频轨道的 `MediaStreamTrack` 对象。

接着，Alice 的应用程序会创建一个新的 `RTCPeerConnection` 对象，这个对象将负责管理与 Bob 连接的整个生命周期。然后，通过调用 `pc.addTrack()` 方法，将从 `MediaStream` 中获取的音视频轨道添加到这个连接实例中，为发送给 Bob 做准备。



### 2.2 阶段二：信令握手 (SDP 提议/应答交换)

SDP 提议/应答交换和 ICE 候选者交换并非两个完全独立的顺序过程，而是一个紧密耦合、相互依赖的状态机。信令服务器在这个过程中扮演着“编舞者”的角色，协调着两个并行流程的进行。

1. **Alice 创建提议 (Offer)：** Alice 的浏览器调用 `pc.createOffer()` 方法。这将异步生成一个 SDP `offer`，其中包含了她期望的会话配置（如支持的编解码器等）。

2. **Alice 设置本地描述：** Alice 将上一步生成的 `offer` 传递给 `pc.setLocalDescription()`。这个操作会告知她本地的 `RTCPeerConnection` 实例其自身的配置。

   **这是一个关键步骤，因为它会立即在后台触发 ICE 候选者的收集过程** 。

3. **Alice 将提议发送给 Bob：** Alice 的应用程序通过预先建立的信令服务器，将 SDP `offer` 的文本内容发送给 Bob。

4. **Bob 接收提议：** Bob 的应用程序从信令服务器接收到 `offer`，并将其传递给 `pc.setRemoteDescription()`。这个操作让 Bob 的 `RTCPeerConnection` 实例了解了 Alice 提议的会话配置。

5. **Bob 创建应答 (Answer)：** 在了解了 Alice 的能力后，Bob 的浏览器调用 `pc.createAnswer()` 来生成一个与 `offer` 兼容的 SDP `answer` 。

6. **Bob 设置本地描述：** Bob 将生成的 `answer` 传递给他自己的 `pc.setLocalDescription()`。这个操作配置了他这一端的连接，并且同样会触发他本地的 ICE 候选者收集过程。

7. **Bob 将应答发送给 Alice：** Bob 通过信令服务器将他的 SDP `answer` 发回给 Alice。

8. **Alice 接收应答：** Alice 收到 `answer` 后，调用 `pc.setRemoteDescription()`。至此，双方都有了一份完整的、协商一致的会话描述。



### 2.3 阶段三：建立连接 (ICE 候选者交换)

得益于 Trickle ICE，这个阶段与 SDP 交换是并行进行的，极大地加快了连接速度。

1. **收集候选者：** 在 Alice 和 Bob 各自调用 `setLocalDescription` 之后，他们本地的 ICE 代理（ICE Agent）便开始工作，收集所有可能的网络路径候选者（主机、服务器反射和中继类型）。
2. **`onicecandidate` 事件：** 每当 ICE 代理发现一个新的候选者，`RTCPeerConnection` 实例就会触发一个 `onicecandidate` 事件。应用程序需要监听这个事件，并从事件对象中获取候选者信息。
3. **通过信令“涓滴”发送：** 事件处理函数会立即将这个新发现的候选者通过信令服务器发送（即“涓滴”）给对端。
4. **添加远端候选者：** 当一方从信令服务器收到对端的候选者时，它会调用 `pc.addIceCandidate()` 方法将这个候选者添加到 `RTCPeerConnection` 实例中。这样，每一方都会建立一个包含所有可能连接到对端的路径列表。
5. **连通性检查：** 与此同时，双方的 ICE 代理会开始对本地和远端候选者组成的“候选者对”（Candidate Pair）进行连通性检查。它们通过互相发送 STUN 绑定请求来测试路径是否可用，并会优先测试成功率更高的路径对（例如，首先是局域网内的主机地址，其次是 STUN 发现的公网地址，最后才是 TURN 中继地址）。



### 2.4 阶段四：加密信道与媒体流传输

1. **路径选择：** 一旦 ICE 代理找到了一个可以成功通信的候选者对，并且连通性检查通过，该路径就会被选定为本次通信的活动路径。此时，`RTCPeerConnection` 的 `iceConnectionState` 状态会变为 `connected` 。
2. **DTLS 握手：** 接下来，双方会通过选定的连接路径执行一次 DTLS（数据报传输层安全）握手。这个过程是强制性的，用于验证对方身份、协商加密算法并生成用于媒体加密的密钥。WebRTC 不允许未加密的通信。
3. **SRTP 媒体流：** DTLS 握手成功完成后，媒体数据就可以开始传输了。音视频数据会使用 DTLS 协商出的密钥，通过 SRTP（安全实时传输协议）进行加密和传输。此时，`RTCPeerConnection` 的 `connectionState` 属性会转换到最终的 `connected` 状态，标志着一个安全、实时的 P2P 连接已成功建立。





## 第 3 节：WebRTC API 实践

本节将理论与实践相结合，为开发者提供开始使用 WebRTC 所需的核心 API 知识和代码框架。



### 3.1 `RTCPeerConnection` API：核心接口

`RTCPeerConnection` 是 WebRTC 的核心接口，它封装了建立和管理对等端连接的所有复杂性。



#### 构造函数与配置

通过 `new RTCPeerConnection(configuration)` 来创建一个新的连接实例。`configuration` 是一个可选的对象，其中最重要的参数是 `iceServers` 数组。该数组用于指定应用可以使用的 STUN 和 TURN 服务器的地址及认证凭据。

```javascript
// WebRTC 连接配置：建议提供 STUN，并为生产环境准备 TURN 兜底
// - STUN：仅用于发现公网可达地址，不中继媒体，成本低
// - TURN：在直连失败时中继媒体，需规划带宽与全球部署
const configuration = {
  iceServers: [
    // 公共 STUN（示例）；生产建议自建或使用可信托管服务
    { urls: ["stun:stun.l.google.com:19302"] },
    // 生产环境通常需要配置 TURN（UDP/TCP/TLS 三栈）作为兜底：
    // { urls: ["turn:turn.example.com:3478"], username: "user", credential: "pass" },
    // { urls: ["turns:turn.example.com:5349"], username: "user", credential: "pass" },
  ],
  // 可选：预聚合 ICE 候选，通常保持 0 使用 Trickle ICE 实时收集
  // iceCandidatePoolSize: 0,
};
const pc = new RTCPeerConnection(configuration);
```



#### 协商相关的关键方法

以下是在提议/应答和 ICE 流程中必须使用到的核心方法：

- `createOffer()` / `createAnswer()`: 异步方法，用于创建 SDP 描述。它们返回一个 Promise，该 Promise 会在 SDP 生成后解析。
- `setLocalDescription(description)` / `setRemoteDescription(description)`: 用于将本地和远端的 SDP 描述应用到连接上。它们同样返回 Promise。
- `addTrack(track, stream)`: 将一个媒体轨道（通常来自 `getUserMedia`）添加到连接中，以便发送给对端。
- `addIceCandidate(candidate)`: 将通过信令从远端接收到的 ICE 候选者提供给本地 ICE 代理。



#### 必不可少的事件处理器

开发者需要为 `RTCPeerConnection` 实例注册几个关键的事件监听器，以驱动和响应连接过程：

- `onnegotiationneeded`: 当发生需要进行新一轮 SDP 协商的变更时（例如，首次添加媒体轨道），会触发此事件。最佳实践是在这个事件的处理函数中调用 `createOffer()` 来发起协商。
- `onicecandidate`: 当本地 ICE 代理发现一个新的候选者时触发。这是将候选者通过信令服务器发送给对端的关键钩子。如果事件的 `candidate` 属性为 `null`，则表示候选者收集已完成。
- `ontrack`: 当从远端接收到一个新的媒体轨道时触发。此事件的处理函数负责将接收到的媒体流附加到一个 `<video>` 或 `<audio>` 元素上进行播放。



#### 监控连接状态

为了给用户提供准确的 UI 反馈并进行有效的调试，监控连接状态至关重要。`RTCPeerConnection` 提供了几个状态属性：

- `signalingState`: 追踪 SDP 提议/应答交换的状态（如 `stable`, `have-remote-offer` 等）。
- `iceGatheringState`: 追踪 ICE 候选者收集的过程（如 `new`, `gathering`, `complete`）。
- `connectionState`: 提供一个高级的、聚合的连接状态，是应用程序逻辑最常使用的状态。它综合了 ICE 和 DTLS 的状态，可以清晰地指示连接的当前情况。

| `connectionState` 状态 | 描述                                                         | 典型触发/应用响应                                            |
| ---------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `new`                  | `RTCPeerConnection` 已创建，但尚未开始连接。                 | 初始状态。等待协商开始。                                     |
| `connecting`           | ICE 代理正在尝试建立连接，至少一个传输通道处于 `checking` 或 `connected` 状态。 | 连接正在进行中。UI 可显示“连接中...”。                       |
| `connected`            | 所有传输通道都已成功建立连接。ICE 连通性检查和 DTLS 握手均已完成。 | 连接成功。媒体流应该已经开始或即将开始。UI 可显示“已连接”。  |
| `disconnected`         | 至少一个传输通道意外断开。这可能是暂时的网络波动，连接或许可以自动恢复。 | 连接中断。应用可以等待一小段时间看是否能自动恢复，或提供手动重连选项。 |
| `failed`               | ICE 或 DTLS 协商彻底失败，连接无法建立或恢复。               | 连接失败。应终止通话，清理资源，并向用户显示错误信息。       |
| `closed`               | 连接已被 `pc.close()` 方法关闭。                             | 通话结束。所有相关资源应被释放。                             |



### 3.2 代码框架：构建一个简单的一对一视频通话



以下是一个实现简单一对一视频通话的基本代码框架。



#### HTML 结构

```html
<!DOCTYPE html>
<html>
<head>
   <title>WebRTC 1-on-1 Video Call</title>
</head>
<body>
<h1>WebRTC Demo</h1>
<div>
   <h2>Local Video</h2>
   <video id="localVideo" autoplay muted playsinline></video>
</div>
<div>
   <h2>Remote Video</h2>
   <video id="remoteVideo" autoplay playsinline></video>
</div>
<button id="startButton">Start</button>
<button id="callButton">Call</button>
<button id="hangupButton">Hang Up</button>
</body>
</html>
```



#### JavaScript 实现

以下代码框架展示了从媒体捕获到连接建立的完整生命周期。



```JavaScript
// 假设的信令通道实现 (在真实应用中，这会是 WebSocket 或其他通信机制)
// 这是一个事件驱动的模拟对象，用于演示信令逻辑
class SignalingChannel extends EventTarget {
   send(message) {
      console.log('Sending message:', message);
      // 在真实应用中，这里会通过 WebSocket 发送 JSON.stringify(message)
      // 为了演示，我们在本地模拟“远端接收”，并触发事件与 onmessage 回调
      setTimeout(() => {
         // 事件派发（推荐）：允许使用 addEventListener('message', ...)
         this.dispatchEvent(new MessageEvent('message', { data: message }));
         // 兼容属性式回调：允许使用 signaling.onmessage = (event) => {...}
         if (typeof this.onmessage === 'function') {
           this.onmessage(new MessageEvent('message', { data: message }));
         }
      }, 100);
   }
}

const signaling = new SignalingChannel();

// DOM 元素
const startButton = document.getElementById('startButton');
const callButton = document.getElementById('callButton');
const hangupButton = document.getElementById('hangupButton');
const localVideo = document.getElementById('localVideo');
const remoteVideo = document.getElementById('remoteVideo');

// WebRTC 变量
let localStream;
let pc;
const configuration = { iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] };
// 建议的媒体约束：提升语音通话质量并控制初始视频档位
// - 音频：开启回声消除、降噪与自动增益
// - 视频：设置理想分辨率与帧率，避免过高码率导致连通性初期不稳定
const constraints = {
  audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
  video: { width: { ideal: 1280 }, height: { ideal: 720 }, frameRate: { ideal: 30, max: 30 } },
};

// 按钮事件监听
startButton.onclick = start;
callButton.onclick = call;
hangupButton.onclick = hangup;

// 1. 开始：获取本地媒体流
async function start() {
   console.log('Requesting local stream');
   startButton.disabled = true;
   try {
      localStream = await navigator.mediaDevices.getUserMedia(constraints);
      localVideo.srcObject = localStream;
      callButton.disabled = false;
   } catch (e) {
      console.error('getUserMedia() error: ', e);
   }
}

// 2. 呼叫：创建 PeerConnection 并发起提议
async function call() {
   callButton.disabled = true;
   hangupButton.disabled = false;
   console.log('Starting call');

   pc = new RTCPeerConnection(configuration);

   // 将本地媒体轨道添加到 PeerConnection
   localStream.getTracks().forEach(track => pc.addTrack(track, localStream));

   // 监听 ICE 候选者事件（Trickle ICE）
   pc.onicecandidate = event => {
      // 候选者会在收集到后被“涓滴”发送给对端
      // 当 candidate 为 null 时，表示本端候选收集完成，可发送结束标志
      signaling.send({ candidate: event.candidate || null });
   };

   // 监听远端媒体轨道事件
   pc.ontrack = event => {
      // event.streams 是数组；常见场景只需取第一路流
      const [remoteStream] = event.streams;
      if (remoteVideo.srcObject !== remoteStream) {
        remoteVideo.srcObject = remoteStream;
        console.log('Received remote stream');
      }
   };

   // 监听连接状态变化
   pc.onconnectionstatechange = (event) => {
      console.log(`Connection state change: ${pc.connectionState}`);
   };

   // 可选：使用 onnegotiationneeded 触发“完美协商”（避免在状态不稳定时重复发起）
   // 提示：当前示例已在下方主动发起了一次 createOffer；若开启本段，需去掉下方主动发起逻辑
   // pc.onnegotiationneeded = async () => {
   //   try {
   //     if (pc.signalingState !== 'stable') return; // 防止在 glare 或重入时重复发起
   //     const offer = await pc.createOffer();
   //     await pc.setLocalDescription(offer);
   //     signaling.send({ desc: pc.localDescription });
   //   } catch (e) {
   //     console.error('onnegotiationneeded failed: ', e);
   //   }
   // };

   // 创建提议 (Offer) 并触发协商
   try {
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      // 将提议通过信令服务器发送给对方
      // 注意：使用 pc.localDescription 可确保发送的是可能被封装后的描述
      signaling.send({ desc: pc.localDescription });
   } catch (e) {
      console.error('Failed to create session description: ', e);
   }
}

// 3. 挂断
function hangup() {
   console.log('Ending call');
   pc.close();
   pc = null;
   hangupButton.disabled = true;
   callButton.disabled = false;
}

// 4. 处理信令消息
// 可使用 addEventListener 或 onmessage；这里保留 onmessage 以便示例最简
signaling.onmessage = async (event) => {
   // 在真实应用中，这里会解析从 WebSocket 收到的消息
   // const message = JSON.parse(event.data);
   const { desc, candidate } = event.data; // 模拟直接接收对象

   try {
      if (desc) {
         // 如果是提议 (Offer)，创建应答 (Answer)
         if (desc.type === 'offer') {
            await pc.setRemoteDescription(desc);
            // 在接收方，也需要获取本地媒体流并添加到连接中
            // (在这个简化的单页面示例中，我们假设 localStream 已经存在)
            // if (!localStream) await start();
            // localStream.getTracks().forEach(track => pc.addTrack(track, localStream));

            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            signaling.send({ desc: pc.localDescription });
         } else if (desc.type === 'answer') {
            // 如果是应答 (Answer)，设置远端描述
            await pc.setRemoteDescription(desc);
         } else {
            console.log('Unsupported SDP type.');
         }
      } else if (candidate !== undefined) {
         // 如果是 ICE 候选者，添加到 PeerConnection
         // candidate 为 null 时表示对端候选收集完毕，可选择调用 end-of-candidates 标志
         if (candidate) {
           await pc.addIceCandidate(candidate);
         } else {
           // 可选：结束信号；某些实现会自动处理
           // await pc.addIceCandidate(null);
         }
      }
   } catch (e) {
      console.error('Error handling signaling message: ', e);
   }
};
```



#### 与真实信令服务器集成

上述代码中的 `signaling` 对象是一个概念性的占位符。在实际应用中，需要将其替换为一个真实的信令服务器连接。最常见的方式是使用 WebSocket。例如，初始化部分会变成：

```js
const socket = new WebSocket('wss://your-signaling-server.com');

// 发送消息
function sendSignalingMessage(message) {
   socket.send(JSON.stringify(message));
}

// 接收消息
socket.onmessage = async (event) => {
   const message = JSON.parse(event.data);
   //... 之后是处理 desc 和 candidate 的逻辑
};
```

应用程序需要定义一套简单的消息格式，以便在对等端之间区分 `offer`、`answer` 和 `candidate` 消息，并可能包含房间号或用户 ID 等用于路由的信息。



## 结论：实时 Web 应用的未来

WebRTC 是一项功能强大的技术，它通过提供一套标准化的原生 API，成功地将无插件、强制加密（DTLS/SRTP）的点对点实时通信能力赋予了 Web 平台。对于真正的端到端加密需求，可结合 Insertable Streams/SFrame 在应用层实现。它不仅是现代视频会议、在线协作和即时通讯应用的基石，也为去中心化网络和物联网等前沿领域开辟了新的可能性。

然而，这种强大能力的背后是显著的架构复杂性。开发者必须深刻理解，WebRTC 并非一个即插即用的解决方案，而是需要精心设计和部署配套基础设施的框架。一个健壮的 WebRTC 应用离不开一个高效、可靠的信令服务器，以及一个能够应对各种复杂网络环境的、包含 STUN 和 TURN 服务器的 NAT 穿透解决方案。
