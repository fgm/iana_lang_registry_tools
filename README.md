# IANA Language registry tools

The code in this repo downloads and parses language subtag information from the IANA language registry at https://www.iana.org/assignments/language-subtag-registry/language-subtag-registry

## YAML format

The results look like this, showcasing the different available fields and subtag types.

```yaml
filedate: "2022-08-08"
entries:
      - added: "2005-10-16"
        description:
          - Church Slavic
          - Church Slavonic
          - Old Bulgarian
          - Old Church Slavonic
          - Old Slavonic
        subtag: cu
        type: language
        # ... 
      - added: "2005-10-16"
        comments: sr, hr, bs are preferred for most modern uses
        description:
          - Serbo-Croatian
        scope: macrolanguage
        subtag: sh
        type: language
        # ... 
      - added: "2009-07-29"
        description:
          - Algerian Saharan Arabic
        macro-language: ar
        preferred-value: aao
        prefix:
          - ar
        subtag: aao
        type: extlang
        # ...
      - added: "2001-11-11"
        description:
          - Lojban
        preferred-value: jbo
        tag: art-lojban
        type: grandfathered
        # ...      
      - added: "2003-05-30"
        description:
          - Azerbaijani in Arabic script
        tag: az-Arab
        type: redundant
        # ...
      - added: "2009-07-29"
        description:
          - Ascension Island
        subtag: AC
        type: region
        # ...
      - added: "2005-10-16"
        description:
          - Latin (Fraktur variant)
        subtag: Latf
        type: script
        # ...
      - added: "2007-03-20"
        comments: 17th century French, as catalogued in the "Dictionnaire de l'académie françoise", 4eme ed. 1694; frequently includes elements of Middle French, as this is a transitional period
        description:
          - Early Modern French
        prefix:
          - fr
        subtag: 1694acad
        type: variant

```
## Changelog

- Initial version: 
  - download, parse and serialize to YAML
  - uses a file cache to avoid downloading every time
