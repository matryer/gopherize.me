$(function(){
	var results = null;
	$("#grid").each(function(){
		var $this = $(this)
		$.ajax({
			url: 'https://gopherize.me/gophers/recent/json?limit=1000',
			success: function(results){
				for (var i in results.gophers) {
					if (!results.gophers.hasOwnProperty(i)) { continue }
					var gopher = results.gophers[i]
					$this.append(
						$('<a>', {href:'/gopher/'+gopher.id}).append(
							$('<img>', {src: gopher.thumbnail_url})
						)
					)
				}
			},
			error: function(){
				console.warn(arguments)
			}
		})
	})
})