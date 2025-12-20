function getBackgroundBrightness(element) {
    var colour = getComputedStyle(element).getPropertyValue("background-color");
    var depth = 0;
    while((colour == "transparent" || colour == "rgba(0, 0, 0, 0)") && depth <= 4)
    {
        element = element.parentNode;
        colour = getComputedStyle(element).getPropertyValue("background-color");
        ++depth;
    }
	var rgb = colour.slice(4, -1).split(", ");
	var result = Math.sqrt(rgb[0] * rgb[0] * .241 +
	rgb[1] * rgb[1] * .691 +
	rgb[2] * rgb[2] * .068);
    return result;
}

function setSpriteLightness(spriteElement)
{
    if(getBackgroundBrightness(spriteElement) < 127)
    {
        spriteElement.classList.add("dark");
    }
    else
    {
        spriteElement.classList.remove("dark");
    }
}

function elementIsFocused(Element)
{
    var Result = false;
    if(Element.classList.contains("focused"))
    {
        Result = true;
    }
    while(Element.parent)
    {
        Element = Element.parent;
        if(Element.classList.contains("focused"))
        {
            Result = true;
            break;
        }
    }
    return Result;
}

function focusSprite(Element)
{
    if(Element.classList.contains("cineraSprite"))
    {
        setSpriteLightness(Element);
        Element.style.backgroundPositionY = Element.getAttribute("data-y-focused") + "px";
    }
    for(var i = 0; i < Element.childElementCount; ++i)
    {
        focusSprite(Element.children[i]);
    }
}

function enableSprite(Element)
{
    if(Element.classList.contains("focused"))
    {
        focusSprite(Element);
    }
    else
    {
        if(Element.classList.contains("cineraSprite"))
        {
            setSpriteLightness(Element);
            Element.style.backgroundPositionY = Element.getAttribute("data-y-normal") + "px";
        }
        for(var i = 0; i < Element.childElementCount; ++i)
        {
            enableSprite(Element.children[i]);
        }
    }
}

function disableSprite(Element)
{
    if(Element.classList.contains("cineraSprite"))
    {
        setSpriteLightness(Element);
        Element.style.backgroundPositionY = Element.getAttribute("data-y-disabled") + "px";
    }
    for(var i = 0; i < Element.childElementCount; ++i)
    {
        disableSprite(Element.children[i]);
    }
}

function unfocusSprite(Element)
{
    if(Element.classList.contains("off"))
    {
        disableSprite(Element);
    }
    else
    {
        if(Element.classList.contains("cineraSprite"))
        {
            setSpriteLightness(Element);
            Element.style.backgroundPositionY = Element.getAttribute("data-y-normal") + "px";
        }
        for(var i = 0; i < Element.childElementCount; ++i)
        {
            unfocusSprite(Element.children[i]);
        }
    }
}
