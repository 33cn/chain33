#!/bin/bash
cli seed save -s "crew clown absent desert grit nerve whale toilet clean action point wall actress equip describe" -p "litian"

cli wallet unlock -p litian

cli account import_key -k "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944" -l test1

cli account create -l test001
