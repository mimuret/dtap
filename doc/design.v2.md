# design

## DNSTAP frame journey

1. Input plugins are input frames to the input buffer.
2. The main controller fetches frames from the input buffer.
3. Fetched Frames are filtered and modified by global filters.
4. Filtered frames are copied for all output groups. And It is filtered and modified by output group filters. After It pushes to the output group buffer.
5. Output plugins are part of output groups that fetch frames from the output group buffer. And It sends to external. 

## Frame flow

```mermaid
flowchart LR
  SOURCE_1(DNSTAP SOURCE No.1)
  SOURCE_N(DNSTAP SOURCE No.N)
  INPUT_PLUGIN_1[Input Plugin No.1]
  INPUT_PLUGIN_N[Input Plugin No.N]
  COMMON_INPUT_BUFFER[Common Input Buffer]
  COMMON_FILTER[Common Filter]
  OUTPUT_GROUP_FILTER[Output Group Filter]
  OG_1[Output Group Buffer No.1]
  OG_N[Output Group Buffer No.N]
  OP_1_1[Output Group No.1 Output Plugin No.1]
  OP_1_N[Output Group No.1 Output Plugin No.1]
  OP_N_1[Output Group No.N Output Plugin No.1]
  OP_N_N[Output Group No.N Output Plugin No.1]
  External1[External ex. File, nats, fluent...]
  External2[External ex. File, nats, fluent...]
  External3[External ex. File, nats, fluent...]
  External4[External ex. File, nats, fluent...]

  subgraph MainInputLoop
    COMMON_INPUT_BUFFER-->|fetch|COMMON_FILTER
    COMMON_FILTER-->OUTPUT_GROUP_FILTER-->|copy|OG_1 & OG_N
  end

  subgraph OutputGroup1_Plugin1
    OG_1-->|fetch| OP_1_1
    OP_1_1-->External1
  end

  subgraph OutputGroup1_PluginN
    OG_1-->|fetch| OP_1_N
    OP_1_N-->External2
  end

  subgraph OutputGroupN_Plugin1
    OG_N-->|fetch| OP_N_1
    OP_N_1-->External3
  end

  subgraph OutputGroupN_PluginN
    OG_N-->|fetch| OP_N_N
    OP_N_N-->External4
  end

  subgraph Input1
    SOURCE_1-->INPUT_PLUGIN_1;
    INPUT_PLUGIN_1  -->|push|COMMON_INPUT_BUFFER
  end
  subgraph InputN
    SOURCE_N-->INPUT_PLUGIN_N;
    INPUT_PLUGIN_N  -->|push|COMMON_INPUT_BUFFER
  end
```
