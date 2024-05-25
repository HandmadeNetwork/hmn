function rem2px(rem) {    
    return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}
