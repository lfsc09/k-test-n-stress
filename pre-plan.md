# Feature - Add mock function `Random.arrayElement:{['elem1', 'elem2']}:{[probElem1, probElem2]}`

# Feature - Create **Action** functions, functions to execute like the mock function but they can aggregate value to the generator

Would have to create the `pipe` functionality which would take the output from a function to feed the next one.

This action functions would be piped like:

```
{{ BLANK }} # Generates a blank value
{{ NULL:{language} }} # Generates a nullish value for the given language

{{ Person.name | OR_BLANK:{prob} }} # Would overwrite the person name with `value` given a probability
{{ Person.name | OR_NULL:{prob}:{language} }} # Would overwrite the person name with `value` given a probability

{{ SEQLINEAR:{start}:{step} }} # Sequence number build 1 -> 2 -> 3

{{ SEQEXPN:{start}:{step} }} # Sequence number build 1 -> 2 -> 4

{{ Person.name | CACHE_WRITE:{key} }}  # Would cache the person name, under a given `key` by the user
{{ CACHE_READ:{key} }} # Would read a `key` from the cache and use that value
```
