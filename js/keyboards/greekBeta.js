letterMap = new Map([['a/', 'ά'], ['a\\', 'ὰ']]);

keys = Array.from(letterMap.keys());
values = Array.from(letterMap.values());

firstLetter = new Array(0);
secondLetter = new Array(0);

for (var i = 0; i < keys.length; i++) {
    firstLetter.push(keys[i].charAt(0));
    secondLetter.push(keys[i].charAt(1));
}

document.addEventListener("keydown", handleKeydown, false);
function handleKeydown(event) {

    var secondTyped = event.key;

    var target = event.target;
    if (!target.classList.contains("specialKey")) {
        return;
    }

    if (secondLetter.includes(secondTyped) == false) {
        return;
    }

    var offset = target.selectionStart;
    var firstTyped = target.value[offset - 1];

    if (firstLetter.includes(firstTyped) == false) {
        return;
    }

    varLookup = firstTyped + secondTyped
    console.log(varLookup)

    if (keys.includes(varLookup)) {
        console.log("hooray")
        event.preventDefault();
        var beforeChar = target.value.slice(0, (offset - 1));
        var afterChar = target.value.slice(offset);
        target.value = (beforeChar + letterMap.get(varLookup) + afterChar);
        target.selectionStart = offset;
        target.selectionEnd = offset;
    }
}