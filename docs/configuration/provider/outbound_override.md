# Outbound Override

Override options applied to all outbounds in the provider.

### Structure

```json
{
  "tag_prefix": "",
  "tag_suffix": "",

  ... // Dial Fields
}
```

### Fields

#### tag_prefix

Prefix added to the tag of every outbound in the provider.

#### tag_suffix

Suffix added to the tag of every outbound in the provider.

#### Dial Fields

Fields set here override the corresponding fields of outbounds in the
provider, see [Dial Fields](/configuration/shared/dial) for details.

!!! note ""

    `detour` is only applied when the outbound does not have its own `detour`.
