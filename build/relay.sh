#!/usr/bin/env bash


function relay_before(){
    sed -i 's/ForkV7AddRelay.*/ForkV7AddRelay = 2/g' ../types/relay.go
}


function relay_after(){
    git checkout ../types/relay.go
}


function relay(){
    echo "================relayd========================"
}




