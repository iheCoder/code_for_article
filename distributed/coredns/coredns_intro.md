# CoreDNS 深入解析：Kubernetes 服务发现的基石

## 前言：为什么我们需要关注 DNS？

在现代的微服务和云原生架构中，服务实例的生命周期是短暂且动态的。IP 地址的频繁变更使得传统的、基于静态 IP 的服务通信方式难以为继。服务发现（Service Discovery）机制应运而生，它解决了“如何在动态环境中找到并访问目标服务”这一核心问题。

在 Kubernetes (K8s) 世界里，DNS 是实现服务发现的基石。它就像是集群内部的“电话簿”，将人类可读的服务名称（如 `api-gateway.prod.svc.cluster.local`）解析成机器可读的 IP 地址。而自 K8s v1.13 版本以来，**CoreDNS** 已成为官方推荐并默认启用的 DNS 服务器。理解 CoreDNS，就是理解 K8s 服务发现脉络的关键。

本文将深入浅出地介绍 CoreDNS 的核心原理，剖析其在 Kubernetes 全局视角下的角色，并通过详实的示例带你领略其强大与灵活性。

## 一、CoreDNS 在 K8s 中的角色

在 K8s 集群中，CoreDNS 通常以一个或多个 Pod 的形式运行（通常由一个 `Deployment` 管理），并由一个名为 `kube-dns` 的 `Service` 暴露出来。这个 Service 的 ClusterIP 是一个稳定不变的地址，集群中所有其他 Pod 的 DNS 请求都将被导向这里。

![CoreDNS in Kubernetes](https://user-images.githubusercontent.com/1283683/101998820-965b6580-3c94-11eb-8588-62343c58774c.png)

具体来说，`kubelet` 在创建每个 Pod 时，会动态生成一个 `/etc/resolv.conf` 文件，其内容大致如下：

```
nameserver 10.96.0.10
search <namespace>.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

- **`nameserver 10.96.0.10`**: 这里的 IP 地址正是 `kube-dns` Service 的 ClusterIP。它告诉 Pod：“如果你有任何 DNS 查询需求，请发往这个地址”。
- **`search ...`**: 这是 DNS 搜索域列表。当你在 Pod 内尝试访问一个短名称（如 `my-service`）时，系统会按顺序尝试拼接这些后缀来构成一个完全限定域名（FQDN）进行查询。例如，它会依次尝试 `my-service.<namespace>.svc.cluster.local`、`my-service.svc.cluster.local` 等。
- **`options ndots:5`**: 这个选项表示，如果查询的域名中的点（`.`）少于5个，DNS 解析器会先尝试使用 `search` 列表中的后缀，然后再将其作为绝对域名进行查询。这优化了集群内部服务的解析效率。

**总结一下 CoreDNS 的核心职责：**

1.  **集群内部服务解析 (Service Discovery)**: 解析形如 `<service-name>.<namespace>.svc.cluster.local` 的域名到对应的 Service ClusterIP。
2.  **Pod 解析 (Pod DNS)**: 根据配置，可以直接将 Pod 的域名（如 `1-2-3-4.default.pod.cluster.local`）解析到其 Pod IP。
3.  **外部服务解析 (Upstream Forwarding)**: 将对集群外部域名（如 `www.google.com`）的查询请求，转发到预定义的上游 DNS 服务器。
4.  **其他高级功能**: 如基于 `hosts` 文件的自定义解析、重写查询、提供 DNS-based service discovery for headless services (SRV records) 等。

## 二、核心原理：插件驱动的架构

CoreDNS 最大的特点是其**插件链（Plugin Chain）**架构。它本身是一个极简的 DNS 服务器，所有具体的功能都通过插件（Plugins）来实现。这种设计带来了极高的灵活性和可扩展性。

当我们启动 CoreDNS 时，会加载一个名为 `Corefile` 的配置文件。这个文件定义了 CoreDNS 的行为，其核心是告诉 CoreDNS：针对哪个域名（Zone）和端口，应该启用哪些插件，以及它们的执行顺序。

一个典型的 K8s `Corefile`（通常存储在 `coredns` `ConfigMap` 中）如下所示：

```
.:53 {
    errors
    health {
       lameduck 5s
    }
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods insecure
       fallthrough in-addr.arpa ip6.arpa
       ttl 30
    }
    prometheus :9153
    forward . /etc/resolv.conf {
       max_concurrent 1000
    }
    cache 30
    loop
    reload
    loadbalance
}
```

让我们来解读这个配置：

- **`.:53 { ... }`**: 这定义了一个 Server Block。`.` 代表所有域名（root zone），`53` 是标准的 DNS 端口。这意味着这个块内的配置将处理所有发往 53 端口的 DNS 查询。

- **插件链**: 大括号内的每一行都代表一个插件。当一个 DNS 查询到达时，它会像流水线一样依次通过这些插件：
    - `errors`: 捕获处理过程中的错误，并以标准格式返回给客户端。
    - `health`: 启用一个 HTTP 健康检查端点（默认在 `:8080/health`），便于 K8s 的 `livenessProbe` 进行健康检查。
    - `ready`: 在所有插件都加载完毕后，通过一个 HTTP 端点（`:8181/ready`）报告 CoreDNS 已准备好接收流量。
    - `kubernetes`: **这是与 K8s 集成最核心的插件**。
        - 它会连接 K8s API Server，监视（Watch）Service 和 Pod 的变化。
        - 当收到对 `cluster.local`（或其子域）的查询时，它会根据内存中的服务信息生成并返回 DNS 记录。
        - `pods insecure` 允许在没有相应 Service 的情况下，直接通过 Pod IP 解析 Pod。
        - `fallthrough` 表示如果 `kubernetes` 插件无法解析某个查询（例如，它不是一个有效的 K8s 服务名），则将请求传递给下一个插件处理。
    - `prometheus`: 暴露一个 Prometheus 指标端点（`:9153/metrics`），用于监控 DNS 查询延迟、缓存命中率等关键指标。
    - `forward . /etc/resolv.conf`: **处理外部域名查询**。
        - `.` 表示它是一个“捕获所有”的转发器。
        - `/etc/resolv.conf` 指示 CoreDNS 将查询转发到宿主机（CoreDNS Pod 所在节点）的 DNS 解析器。这通常是由云提供商（如 AWS, GCP）或数据中心网络提供的上游 DNS。
    - `cache 30`: 缓存 DNS 响应 30 秒。这极大地降低了对上游 DNS 服务器的请求压力，并加快了重复查询的响应速度。
    - `loop`: 检测并阻止无限循环的 DNS 查询。
    - `reload`: 允许在不中断服务的情况下，通过修改 `Corefile` 的 `ConfigMap` 来热加载配置。
    - `loadbalance`: 对返回的 A、AAAA 或 MX 记录进行轮询（Round Robin），提供基本的客户端负载均衡。

**处理流程总结**：一个对 `my-svc.default.svc.cluster.local` 的查询会先被 `kubernetes` 插件匹配并成功解析。而一个对 `www.google.com` 的查询，`kubernetes` 插件无法处理，于是 `fallthrough` 到 `forward` 插件，后者将其发往上游 DNS 服务器，并将结果通过 `cache` 插件缓存后返回。

## 三、示例详实：一次完整的 DNS 查询之旅

让我们通过一个实际操作，来追踪一次 DNS 查询的全过程。

**1. 准备环境：部署一个应用**

首先，我们部署一个简单的 Nginx 应用及其 Service。

`nginx-app.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.20
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: my-nginx-svc
spec:
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
```

应用它：`kubectl apply -f nginx-app.yaml`

查看 Service，记下它的 ClusterIP：
`kubectl get svc my-nginx-svc`
```
NAME           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
my-nginx-svc   ClusterIP   10.100.200.30   <none>        80/TCP    1m
```

**2. 进入一个客户端 Pod**

为了模拟查询，我们启动一个临时的 `busybox` Pod，并进入它的 shell。

`kubectl run busybox --image=busybox:1.28 --rm -it -- /bin/sh`

> 补充知识：绝对域名 vs. 相对域名
> 
> 在深入查询之前，我们先厘清一个重要概念：**绝对域名（Absolute Domain Name）** 和 **相对域名（Relative Domain Name）**。
> - **绝对域名**，也称为完全限定域名（FQDN），是 DNS 树状结构中一个独一无二、完整的名称，它从根域（`.`）开始。例如，`my-nginx-svc.default.svc.cluster.local.` 是一个绝对域名（最后的点代表根，通常可以省略）。当 DNS 解析器收到一个绝对域名查询时，它会直接进行查询，不再使用任何搜索后缀。
> - **相对域名** 是一个不完整的名称。例如，我们后面会用到的 `my-nginx-svc`。当解析器看到一个相对域名时，它会认为这个域名是相对于某个“环境”的，并使用 `/etc/resolv.conf` 文件中的 `search` 列表来补全它。例如，在 `default` 命名空间的 Pod 中，解析器会依次尝试：
    1. `my-nginx-svc.default.svc.cluster.local.`
    2. `my-nginx-svc.svc.cluster.local.`
    3. `my-nginx-svc.cluster.local.`

这个自动补全机制极大地简化了集群内部的服务访问。

**3. 开始查询**

现在我们在这个 `busybox` Pod 内部执行 DNS 查询。

**a. 查询内部服务**

使用 `nslookup` 工具查询我们刚刚创建的 Service：

```sh
# / # nslookup my-nginx-svc
Server:    10.96.0.10
Address:   10.96.0.10:53

Name:      my-nginx-svc.default.svc.cluster.local
Address:   10.100.200.30
```

**发生了什么？—— 深入 `kubernetes` 插件的匹配过程**

1.  **发起查询**: `busybox` Pod 向 `/etc/resolv.conf` 中定义的 `nameserver` (10.96.0.10，即 CoreDNS) 发送了对相对域名 `my-nginx-svc` 的查询。
2.  **域名补全**: Pod 内的 DNS 解析器根据 `search` 列表，将相对域名 `my-nginx-svc` 补全为绝对域名 `my-nginx-svc.default.svc.cluster.local.`，然后将这个完整的查询发送给 CoreDNS。
3.  **插件链处理**: CoreDNS 接收到请求，请求在插件链中流转。当到达 `kubernetes` 插件时，匹配过程开始。
4.  **区域匹配 (Zone Matching)**: `kubernetes` 插件的核心工作机制是基于区域（Zone）的。在我们的 `Corefile` 中，它被配置为处理 `cluster.local` 这个区域：`kubernetes cluster.local ...`。插件会检查收到的查询域名 `my-nginx-svc.default.svc.cluster.local.` 是否是 `cluster.local` 的一个子域。这是一个简单的字符串后缀匹配，因为查询的域名以 `cluster.local` 结尾，所以匹配成功。**这就是“发现域名匹配它所管理的区域”的具体过程。**
5.  **域名解析**: 匹配成功后，`kubernetes` 插件便“认领”了这个查询，并开始解析它。它按照 ` <service>.<namespace>.svc.<zone>` 的格式来拆解域名：
    - `service`: `my-nginx-svc`
    - `namespace`: `default`
    - `svc`: 表明这是一个 Service 查询
6.  **查询 K8s API**: 插件通过其与 K8s API Server 的连接，在内存缓存中查找 `default` 命名空间下名为 `my-nginx-svc` 的 Service 对象。
7.  **构建响应**: 成功找到后，插件获取到该 Service 的 ClusterIP `10.100.200.30`，并用它构建一个 DNS A 记录，然后将这个响应返回给客户端。查询至此结束，请求不会再传递给 `forward` 等后续插件。

**b. 查询外部服务**

现在，我们查询一个公网域名：

```sh
# / # nslookup www.github.com
Server:    10.96.0.10
Address:   10.96.0.10:53

Non-authoritative answer:
Name:      www.github.com
Address:   20.205.243.166

...
```

**发生了什么？**
1.  `busybox` Pod 向 CoreDNS 发送对 `www.github.com` 的查询。
2.  CoreDNS 的 `kubernetes` 插件发现域名不属于 `cluster.local`，于是通过 `fallthrough` 将请求交给下一个插件。
3.  请求流经 `prometheus` 等插件，最终到达 `forward` 插件。
4.  `forward` 插件将此查询请求转发给 `/etc/resolv.conf` 中定义的上游 DNS 服务器。
5.  上游 DNS 服务器返回了 `www.github.com` 的公网 IP 地址。
6.  CoreDNS 收到响应，`cache` 插件将其缓存，然后返回给 `busybox` Pod。

## 四、总结

CoreDNS 以其简洁、高效和高度可扩展的插件化设计，完美地契合了 Kubernetes 对动态服务发现的需求。它不仅仅是一个简单的 DNS 解析器，更是连接集群内部服务与外部世界的关键桥梁，同时也是保障集群网络稳定性和可观测性的重要组件。

对于任何希望深入理解 Kubernetes 网络模型的开发者或运维工程师来说，掌握 CoreDNS 的工作原理都是必不可少的一步。通过定制 `Corefile` 和利用丰富的插件生态，你可以实现各种复杂的 DNS 策略，从而更好地驾驭你的云原生应用。
