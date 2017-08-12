$(function(){
    
    $.ajax({
        url: '/gophers.json?limit=200',
        datatype: 'json',
        success: function(result) {
            var gophers = shuffle(result.gophers)
            gophers = gophers.slice(0, 10)
            var cards = []
            for (var i in gophers) {
                cards.push({
                    imageUrl: gophers[i].url,
                    linkUrl: 'https://gopherize.me/gopher/' + gophers[i].id
                })
            }
            var sw = $('h2 .time').stopwatch();
            $('#board').memoryGame({
                cards: cards,
                cardWidth: 65*2,
                cardHeight: 70*2,
                minCardMargin: 10,
                maxCardMargin: 25,
                onPairDisclosed: function(e) {
                    if (e.finished) {
                        var time = sw.stopwatch('getTime')
                        location.href = 'https://twitter.com/intent/tweet?text=' + encodeURIComponent("I just matched ten Gophers in " + time + "ms! Can you beat that? https://pairs.gopherize.me/ #gopherizeme #golang via @ashleymcnamara and @matryer" )
                    }
                }
            })
            sw.stopwatch('start')
        }
    })

    function shuffle(o){
        for(var j, x, i = o.length; i; j = Math.floor(Math.random() * i), x = o[--i], o[i] = o[j], o[j] = x);
        return o;
    }

})