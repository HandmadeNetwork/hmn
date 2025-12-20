var cineraDropdownNavigation = document.getElementsByClassName("cineraNavDropdown");
for(var i = 0; i < cineraDropdownNavigation.length; ++i)
{
    var cineraFamily = cineraDropdownNavigation[i].getElementsByClassName("cineraNavHorizontal")[0];
    cineraDropdownNavigation[i].addEventListener("click", function() {
        if(cineraFamily.classList.contains("visible"))
        {
            cineraFamily.classList.remove("visible");
        }
        else
        {
            cineraFamily.classList.add("visible");
        }
    });

    cineraDropdownNavigation[i].addEventListener("mouseenter", function() {
        cineraFamily.classList.add("visible");
    });

    cineraDropdownNavigation[i].addEventListener("mouseleave", function() {
        cineraFamily.classList.remove("visible");
    });
}

var Sprites = document.getElementsByClassName("cineraSprite");

for(var i = 0; i < Sprites.length; ++i)
{
    var This = Sprites[i];

    var TileX = This.getAttribute("data-tile-width");
    var TileY = This.getAttribute("data-tile-height");
    var AspectRatio = TileX / TileY;

    // TODO(matt):  Nail down the desiredness situation. Perhaps respond to:
    //                  width / min-width / height / min-height set in a CSS file
    //                  flexbox layout, if possible

    // NOTE(matt):  These values are "decoupled" here, to facilitate handling of sizes other than the original
    //              We'll probably need some way of checking the desired and original of both the X and Y, and pick which one on
    //              which to base the computation of the other
    var DesiredX = TileX;
    var DesiredY = DesiredX / AspectRatio;
    var Proportion = DesiredX / TileX;
    //
    ////

    // NOTE(matt):  Size the container and its background image
    //
    This.style.width = DesiredX + "px";
    This.style.height = DesiredY + "px";

    var SpriteWidth = This.getAttribute("data-sprite-width");
    var SpriteHeight = This.getAttribute("data-sprite-height");
    This.style.backgroundSize = SpriteWidth * Proportion + "px " + SpriteHeight * Proportion + "px";
    //
    ////

    // NOTE(matt):  Pick the tile
    //
    setSpriteLightness(This);
    if(This.classList.contains("dark"))
    {
        This.style.backgroundPositionX = This.getAttribute("data-x-dark") + "px";
    }

    if(elementIsFocused(This))
    {
        This.style.backgroundPositionY = This.getAttribute("data-y-focused") + "px";
    }

    if(This.classList.contains("off"))
    {
        This.style.backgroundPositionY = This.getAttribute("data-y-disabled") + "px";
    }
    else
    {
        This.style.backgroundPositionY = This.getAttribute("data-y-normal") + "px";
    }
    //
    ////

    // NOTE(matt):  Finally apply the background image
    var URL = This.getAttribute("data-src");
    This.style.backgroundImage = "url('" + URL + "')";
}
