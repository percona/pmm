# MySQL binaries are 32 bits

## Description
Older version MySQL server (plus some custom compilations) might be compiled for 32 bits whereas we suggest using the new 64 bits version. 


## Rule
`Select @@global.version_compile_machine as version_compile_machine == ‘i686’`


## Resolution
Binaries should be re installed but using the 64 bits version of the version binaries used
