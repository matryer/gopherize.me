
(function($){

	if (location.protocol.indexOf('https')===-1) {
		if (location.href.indexOf('fromhttp')===-1) {
			var n = 'https://'+location.href.split('://')[1]
			if (location.href.indexOf('?')===-1) {
				n += '?fromhttp=true'
			} else {
				n += '&fromhttp=true'
			}
			location.href = n
		}
	}

	var apihost = '/api/'
	apihost = 'https://gopherize.me/api/'
	var apiArtwork = apihost + 'artwork/'
	var artworkResponse = null
	var artwork = null

	var selection = []

	function absurl() {
		return apihost+'render.png?dl=0&images=' + encodeURIComponent(selection.join('|'))
	}

	$('#download-button').click(function(){
		var $this = $(this)
		$this.attr("disabled", "disabled")
		location.href = '/api/render.png?images=' + encodeURIComponent(selection.join('|'))
	})

	$('#share-button').click(function(){
		var $this = $(this)
		var absURL = absurl()
		var text = "I just Gopherized myself on https://gopherize.me via @ashleymcnamara and @matryer"
		var shareURL = 'https://twitter.com/share?url='+encodeURIComponent(absURL)+'&text='+encodeURIComponent(text)+'&hashtags=golang,gopherize'
		window.open(shareURL)
	})

	$('#buy-button').click(function(){
		buy()
	})

	$('#next-button').click(function(){
		next()
	})

	function loadArtwork(callback) {
		busy(true)
		$.ajax({
			url: apiArtwork,
			success: callback,
			error: function(){
				console.warn(arguments)
			},
			complete: function(){
				busy(false)
			}
		})
	}

	function busy(is) {
		if (!is) {
			$(".busy").hide()
			return
		}
		$(".busy").show()
	}

	function getImageByID(id) {
		for (var cat in artwork) {
			if (!artwork.hasOwnProperty(cat)) { continue }
			var category = artwork[cat]
			for (var img in category.images) {
				if (!category.images.hasOwnProperty(img)) { continue }
				var image = category.images[img]
				if (image.id === id) {
					return image
				}
			}
		}
		return null
	}

	function updatePreview() {
		$("#next-button").prop("disabled", false)
		var ids = []
		var previewEl = $('#preview').empty()
		var special = true
		$('#options').find('input:checked').each(function(){
			var $this = $(this)
			var id = $this.val()
			var img = getImageByID(id)
			if (img != null) {
				ids.push(id)
				var mt = special ? 0 : -1000
				previewEl.append(
					$("<img>", {src: img.href}).css({
						marginTop: -1000
					})
				)
				special = false
			}
		})
		selection = ids
		var i = 1;
		previewEl.find("img").each(function(){
			var $this = $(this)
			$this.animate({
				marginTop: 0
			}, 250*i)
			i++
		})
	}

	function shuffle() {
		var i = 0;
		for (var cat in artwork) {
			if (!artwork.hasOwnProperty(cat)) { continue }
			i++
			var special = i<3
			var category = artwork[cat]
			var rand = Math.round(Math.random()*(category.images.length+5))-6
			if (rand < 0 && special) {
				rand = 0
			}
			if (rand < 0) {
				// none
				$('input[name="'+category.name+'"]').prop('checked', false)
				continue	
			}
			var image = category.images[rand]
			$('input[value="'+image.id+'"]').prop('checked', true)
		}
		updatePreview()
	}

	function nicename(s) {
		return s.replace(/_/g, ' ')
	}

	function buy() {
		window.open("https://www.zazzle.co.uk/api/create/at-238314746086099847?rf=238314746086099847&ax=DesignBlast&sr=250359396377602696&cg=0&t__useQpc=true&t__smart=true&continueUrl=https%3A%2F%2Fwww.zazzle.co.uk%2Fgopherizemestore&fwd=ProductPage&tc=&ic=&gopher="+encodeURIComponent(absurl()))
	}

	function reset() {
		$("form#options").trigger("reset")
		selection = []
		updatePreview()
	}

	function next() {
		$("#next-button").prop("disabled", true)
		location.href = '/save?images=' + encodeURIComponent(selection.join('|'))
	}

	$(function(){

		$('#shuffle-button').click(function(){
			shuffle()
		})
		$('#reset-button').click(function(){
			reset()
		})

		var optionsEl = $('#options')

		loadArtwork(function(result){
			artworkResponse = result
			artwork = result.categories
			$(".total_combinations").text(Humanize.intComma(artworkResponse.total_combinations) + " possible combinations")
			var i = 0
			var special = true
			for (var cat in artwork) {
				if (!artwork.hasOwnProperty(cat)) { continue }
				i++
				special = i<3
				var category = artwork[cat]
				var catID = category.name
				var list = $("<div>")
				
				if (!special) {
					$("<label>", {class:'none item'}).append(
						$('<input>', {type:'radio', name:catID, value: "<none>", checked: (special ? 'checked' : null)}).change(updatePreview),
						$('<img>', {src: "/static/whitebox.png", 'title':'Remove', 'data-toggle':'tooltip', 'data-placement':'bottom'}).tooltip()
					).appendTo(list)
				}

				var specialInCat = true
				for (var img in category.images) {
					if (!category.images.hasOwnProperty(img)) { continue }
					var image = category.images[img]

					$("<label>", {class:'item'}).append(
						$('<input>', {type:'radio', name:catID, value:image.id, checked: (special && specialInCat ? 'checked' : null)}).change(updatePreview),
						$('<img>', {src: image.thumbnail_href, 'title':image.name, 'data-toggle':'tooltip', 'data-placement':'bottom'}).tooltip()
					).appendTo(list)
					specialInCat = false

				}
				
				var panel = $("<div>", {class:'panel panel-default'})
				panel.append(
					$("<div>", {class:'panel-heading', role:'tab'}).append(
						$("<h4>", {class:'panel-title'}).append(
							$("<a>", {
								'class': (special ? '' : 'collapsed'),
								'role': 'button',
								'data-toggle': 'collapse',
								'data-parent': '#options',
								'href': '#'+catID,
								'aria-expanded': (special ? 'true' : 'false'),
								'aria-controls': catID
							}).text(nicename(category.name)).tooltip()
						)
					)
				)
				panel.append(
					$("<div>", {id:catID, class:'panel-collapse collapse' + (special ? ' in' : ''), role:'tabpanel'}).append(
						$("<div>", {class:'panel-body'}).append(
							list
						)
					)
				)

				optionsEl.append(panel)

			}

			updatePreview()

		})

	})

})(jQuery)