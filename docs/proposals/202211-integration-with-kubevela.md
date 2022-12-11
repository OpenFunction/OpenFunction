# 我们可以用 KubeVela 做什么

## 什么是 KubeVela ？

KubeVela 是一个基于 Kubernetes 的应用交付和管理平台。它实现了 Kubernetes 在应用层面的能力抽象，并提供了跨云多集群环境的应用持续交付的能力。

KubeVela 以应用为核心向用户提供可编程的交付工作流程，其主要组件（概念）如下：(参考 [KubeVela core-concept](https://kubevela.io/docs/next/getting-started/core-concept))

- **组件（Component）**: 组件定义一个应用包含的待交付制品（二进制、Docker 镜像、Helm Chart...）或云服务。我们认为一个应用部署计划部署的是一个微服务单元，里面主要包含一个核心的用于频繁迭代的服务，以及一组服务所依赖的中间件集合（包含数据库、缓存、云服务等），一个应用中包含的组件数量应该控制在约 15 个以内。
- **运维特征（Trait）**: 运维特征是可以随时绑定给待部署组件的、模块化、可拔插的运维能力，比如：副本数调整（手动、自动）、数据持久化、 设置网关策略、自动设置 DNS 解析等。
- **应用策略（Policy）**: 应用策略负责定义指定应用交付过程中的策略，比如多集群部署的差异化配置、资源放置策略、安全组策略、防火墙规则、SLO 目标等。
- **工作流步骤（Workflow Step）**: 工作流由多个步骤组成，允许用户自定义应用在某个环境的交付过程。典型的工作流步骤包括人工审核、数据传递、多集群发布、通知等。

你可以通过 KubeVela 提供的 [Quick Start](https://kubevela.io/docs/next/quick-start) 部署你的第一个应用。

## OpenFunction 与 KubeVela

KubeVela 可以帮助 OpenFunction 提升以下几方面的能力：

- 进一步优化 OpenFunction 的部署方式
  - 将 OpenFunction 以 addon 形式注册到 KubeVela 的 addon registry 中
  - 可以通过 addon 的形式管理 OpenFunction 的依赖组件（当前 KubeVela 官方已有 [Dapr addon](https://github.com/kubevela/catalog/tree/master/experimental/addons/dapr)），实现更友好的 enable\disable 控制
  - 可以增强在多集群、多环境中的部署能力，减少用户的适配成本

- 提供面向用户的应用级资源管理接口
  - 将 `Function`、`Builder`、`Serving` 等资源添加到 Component 中，使得用户可以在自己的 OAM Applications 中便捷使用
  - 提供相关 Trait、Policy 资源的支持，以适应复杂环境中的工作负载变化


## 设想的使用案例

假设我们向新用户推广 OpenFunction 作为企业内部 Serverless 平台解决方案，实施的步骤大体如下：

1. 部署 KubeVela 并准备以下 addon：
   - FluxCD
   - Cert Manager
   - 依赖组件 addon（即我们需要提供的 addon）
   - OpenFunction（即我们需要提供的 addon）
2. 用 Cert Manager 插件在组件中部署 Cert Manager
3. 用依赖组件 addon 完成依赖组件的部署
4. 用 OpenFunction addon 在组件中部署 OpenFunction
5. 支持在 OAM Applications 定义中使用 `Function`、`Builder`、`Serving` 类型的 Component，以完成特定的 Serverless 平台（应用）建设
6. 在工作流中完成基础资源的准备（如 ConfigMap、Secrets、Ingress 等）
7. 暂停部署步骤，执行一个简单的函数演示，等待用户验证
8. 待用户验证后，手动恢复部署步骤
9. 随后在正式环境中完成整体平台（应用）的建设（过程中会根据 Policy 完成运维特征属性的变更）

## Addon

可以参考 KubeVela 官方文档开发 OpenFunction addon：[Build your Own Registry](https://kubevela.io/docs/platform-engineers/addon/addon-registry) 。

## 总结

通过 KubeVela，我们可以向用户弱化 OpenFunction 的底层实现和依赖组件运维，让他们专注于 OpenFunction 的 Serverless 特性。同时，这种方法增强了 OpenFunction 的产品维度，有利于其独立交付。