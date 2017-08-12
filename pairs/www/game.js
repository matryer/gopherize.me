/* MemoryGame: a jQuery plugin for creating memory card games
               by Matteo Sisti Sette

https://github.com/matteosistisette/jquery-ui-memory-game

*/

(function($){
	
	$.fn.animateScaleX = function(startScale, endScale, duration, easing, complete) {
	    var args = $.speed(duration, easing, complete);
	    var step = args.step;
	    return this.each(function(i, e) {
			args.step = function(now) {
				$.style(e, 'transform', 'scaleX(' + now + ')');
				if (step) return step.apply(this, arguments);
			};
			args.complete=function(){
				if (complete) return complete.apply(e,arguments);
			};
			$({dummyscalex: startScale}).animate({dummyscalex: endScale}, args);
	    });
	};
	
	$.widget("matteosistisette.memoryGame", {
		
		disclosed: null, 		//[]
		ndisclosed: 0,
		elementId: "",
		currentCards: null,		//[]
		timeoutId: null,
		ready: false,
		innerElement: null,
		resetPending: null,
		nmoves: 0,
				
		options: {
			
			cards: [],		
			
			imagesPath: "",
			
			cardWidth: 'auto',
			cardHeight: 'auto',
			
			preferredAspectRatio: 1,
			autoResize: true,
			
			cardFlipDuration: 300, 
			flipBackTimeout: 1000,
			flipAnimationEasing: "linear",
			
			minCardMargin: 10,
			maxCardMargin: 50,
			
			maxRotation: 10,
			
			order: null,
			alreadyDisclosed: [],
		
			onPairDisclosed: function(object) {}
			
		},
		
		reset: function(animated, reorder, rebuild) {
			// Covers all cards.
			// if (reorder) reorders them randomly
			// if (rebuild) completely rebuilds the widget
			//              (usually only needed after modifying the cards array)
			
			if (animated) {
				this.resetPending={
					reorder: reorder,
					rebuild: rebuild
				};
				this._closeCards();
			}
			else {
				this.resetPending=null;
				if (rebuild) {
					this._cleanUp();
					this._build();
				}
				else {
					this._resetCards();
					if (reorder) this.reorder(true);
				}
			}
		},
		
		reorder:function(keepRotations, useOptions) {
			if (keepRotations===undefined) keepRotations=true;
			if (this.options.order===null || this.options.order.length!=2*this.options.cards.length) useOptions=false;
			if (useOptions) keepRotations=false;
			
			var cards=[];
			var rotations=[];
			$(this.innerElement).children().each(function(){
				$(this).detach();
				if (useOptions) {
					var idx=$(this).data("cardIndex");
					if (cards[idx]===undefined) cards[idx]=[];
					cards[idx].push($(this));
				}
				else {
					cards.push($(this));
					rotations.push($(this).find(".memory-card-wrapper").data("rotation"));
				}
			});
			if (useOptions) {
				for (var i=0; i<this.options.order.length; i++) {
					var $card=cards[this.options.order[i]].pop();
					$(this.innerElement).append($card);
				}
			}
			else {
				var i=0;
				while (cards.length>0) {
					var n=Math.min(Math.floor(Math.random()*cards.length),cards.length-1);
					var $card=cards[n];
					cards.splice(n,1);
					if (keepRotations) {
						$card.find(".memory-card-wrapper").data("rotation", rotations[i]).css({
							transform: 'rotate('+rotations[i]+'deg)'
						});
					}
					$(this.innerElement).append($card);
					i++;
				}
			}
		},
		
		rearrange: function() {
			// Sets rotations, positions, etc.
			// Shouldn't be needed as a public method.
			//   Needed after changing some size options, 
			//   but it will be done internally already.
			this.resize();
			this._arrangeCards();
			
		},
				
		destroy: function () {
			this._cleanUp();
			$.Widget.prototype.destroy.call(this);
		},
		
		resize: function (ncalls) {
			if (ncalls===undefined) ncalls=0;
			var done=false;
			
			var gameWidth=$(this.element).width()-1;
			var maxHeight=this._getMaxHeight();
			
			// No of columns using full width with minimum margin:
			var ncards=this.options.cards.length*2;
			var columns = Math.min(Math.floor(gameWidth/(this.options.cardWidth+2*this.options.minCardMargin)), ncards);
			var rows=Math.ceil(ncards/columns);
			
			if (this.options.preferredAspectRatio>0 && rows*(this.options.cardHeight+2*this.options.minCardMargin)<maxHeight) {
				var squareColumns=Math.ceil(Math.sqrt(ncards*this.options.preferredAspectRatio));
				if (squareColumns<columns) {
					var squareRows=Math.ceil(ncards/squareColumns);
					if (squareRows*(this.options.cardHeight+2*this.options.minCardMargin)<=maxHeight) {
						columns=squareColumns;
						rows=squareRows;
					}
					else {
						var maxRows=Math.floor(maxHeight/(this.options.cardHeight+2*this.options.minCardMargin));
						rows=maxRows;
						columns=Math.ceil(ncards/rows);
					}
				}
			}
			
			
			var outerWidth=Math.max(2*this.options.minCardMargin+this.options.cardWidth, Math.min(2*this.options.maxCardMargin+this.options.cardWidth,
				Math.floor(gameWidth/columns)
			));
			
			var outerHeight=this.options.cardHeight+(outerWidth-this.options.cardWidth);
			if (outerHeight*rows>maxHeight) {
				outerHeight=Math.max(Math.floor(maxHeight/rows), this.options.cardHeight+2*this.options.minCardMargin);
			}
			
			var boundWidth=outerWidth*columns;
			if (gameWidth-boundWidth>outerWidth/2) {
				$(this.innerElement).css({
					width: boundWidth
				});
			}
			else {
				$(this.innerElement).css({
					width: "100%"
				});
			}
			
			
			$(this.innerElement).children(".memory-card-container").css({
				width: outerWidth,
				height: outerHeight
			});
			
			var newGameWidth=$(this.innerElement).width()-1;
			if (newGameWidth!=gameWidth) {
				ncalls++;
				if (ncalls<10) {
					this.resize(ncalls);
				}
			}
			
		},
		
		
		_getMaxHeight: function() {
			return document.documentElement.clientHeight;
		},
		
		
		_arrangeCards: function(rotate) {
			var game=this;
			$(this.innerElement).children(".memory-card-container").each(function(){
				var $a=$(this).find("a");
				$a.css({
					marginLeft: -game.option("cardWidth")/2,
					marginTop: -game.option("cardHeight")/2
				});
				$a.find(".card").css({
					width: game.option("cardWidth"),
					height: game.option("cardHeight")
				});
				var $wrapper=$(this).find(".memory-card-wrapper");
				if (rotate) {
					var rotation=(Math.random()*2-1)*game.option("maxRotation");
					$wrapper.css({
						transform: 'rotate('+rotation+'deg)'
					}).data("rotation", rotation);
				}
			});
		},
		
		
		_setOption: function (key,value) {
			this._super(key,value);
		},
		_setOptions: function(options) {
			this._super(options);
			if (this.ready){
			  	if (options.cards!==undefined) {
					this.reset(false, true,true);
				}
				else if (options.cardWidth!==undefined
					|| options.cardHeight!==undefined
					|| options.maxRotation!==undefined
				) {
					this.rearrange();
				}
			}
			
		},
		
		_create: function () {
			this._init(true);
			this._build();
			var game=this;
			if (this.options.autoResize) $(window).resize(function() {
				game.resize();
			});
		},
		
		_init: function(firstTime) {
			this.disclosed=[];
			this.ndisclosed=0;
			this.nmoves=0;
			this.currentCards=[];
			this.elementId=$(this.element).attr("id");
			this.resetPending=null;
			
			if (firstTime) {
				if (this.options.cards.length>0) {
					if (this.options.cardWidth==='auto' || this.options.cardHeight==='auto') {
						throw new Error(
							"Unspecified card size. When passing in a list of cards via the 'cards' parameter, "+
							"you must specify a numeric value for the 'cardWidth' and 'cardHeight' parameters in pixels. "+
							"You cannot use the 'auto' value or not specify it."
						);
					}
					else if (!(Number(this.options.cardWidth)>0) || !(Number(this.options.cardHeight)>0)) {
						throw new Error(
							"Invalid card size."
						);
					}
				}
				if (this.options.cards.length==0 && $(this.element).find("a").length>0) {
					this._createCardsArrayFromMarkup();
				}
				this.ndisclosed=this.options.alreadyDisclosed.length;
				for (var i=0; i<this.options.alreadyDisclosed.length; i++) {
					this.disclosed[this.options.alreadyDisclosed[i]]=true;
				}
			}
		},
		
		_cleanUp: function() {
			$(this.innerElement).children().each(function(){
				$(this).stop(true);
			});
			$(this.element).children().each(function(){
				if ($(this).hasClass("memory-game-inner")) $(this).children().remove();
			}).remove();
			if (this.timeoutId) {
				clearTimeout(this.timeoutId);
				this.timeoutId=null;
			}
			this._init();
		},
				
		_build: function() {
			$(this.element).children().each(function(){
				if ($(this).hasClass("memory-game-inner")) $(this).children().remove();
			}).remove();
			
			this.innerElement=$(this.element).addClass("memory-game").wrapInner('<div class="memory-game-inner"></div>').children()[0];
			
			var cards=this.options.cards;
			for (var i=0; i<cards.length; i++) {
				this._createCard(i, cards[i]);
			}
			this.reorder(true, true);
			this.resize();
			this._arrangeCards(true);
			this.ready=true;
		},
		
		_createCardsArrayFromMarkup: function() {
			var game=this;
			var cards=[];
			this.options.imagesPath="";
			var first=true;
			$(this.element).find("a").each(function(){
				var linkUrl=$(this).attr("href");
				var linkTitle=$(this).attr("title");
				var $img=$(this).find("img");
				if ($img.length>0) {
					if (first) {
						if (game.options.cardWidth==='auto' || game.options.cardHeight==='auto') {
							game.options.cardWidth=$img.width();
							game.options.cardHeight=$img.height();
							if (game.options.cardWidth==0 || game.options.cardHeight==0) {
								throw new Error(
									"Card width and/or height appears to be zero. "+
								    "When parameters cardWidth and cardHeight are not specified or their value is 'auto', "+
								    "you need to make sure that the first image has a measurable width and height when the widget is created, "+
								    "either by creating the widget on document load (as opposed to document ready) or by forcing the image size "+
								    "via 'width' and 'height' html attributes or CSS"
								);
							}
						}
						first=false;
					}
					var imageUrl=$img.attr("src");
					var cardObject={
						linkUrl: linkUrl,
						linkTitle: linkTitle,
						imageUrl: imageUrl,
						data: $(this).data(),
					};
					cards.push(cardObject);
				}
			});
			this.options.cards=cards;
		},
		
		_resetCards: function() {
			if (this.timeoutId) {
				clearTimeout(this.timeoutId);
				this.timeoutId=null;
			}
			this._init(false);
			var game=this;
			$(this.innerElement).children().each(function(){
				if ($(this).data("status")>0) {
					$(this).stop(true);
					game._setCardStatus($(this), 0, true);
				}
			});
		
		},
		
		_closeCards: function() {
			if (this.timeoutId) {
				clearTimeout(this.timeoutId);
				this.timeoutId=null;
			}
			var game=this;
			$(this.innerElement).children().each(function(){
				if ($(this).data("status")>0) {
					if ($(this).data("currentDirection")>=0) {
						game._startFlip(this,-1);
					}
				}
			});
		},
		
		_createCard: function(cardIndex, cardInfo) {
			for (var i=0; i<2; i++) {
				var containerId=this.getCardId(cardIndex,i);		
				var $container=$('<div class="memory-card-container" id="'+containerId+'"></div>');
				var href=cardInfo.linkUrl;
				if (href===undefined || href===null) href="";
				
				var $wrapper=$('<div class="memory-card-wrapper"></div>');
				$container.append($wrapper);
				
				var $a=$('<a href="" target="_blank"></a>');
				$container.data("linkUrl", href);
				$container.data("linkTitle", cardInfo.linkTitle);
				$container.data("cardIndex", cardIndex);
				$container.data("instance", i);
				$container.data("game", this);
				$container.data("currentDirection", 0);
				if (cardInfo.data!==undefined) $a.data(cardInfo.data);
				$wrapper.append($a);
				var $img=$('<img src="'+this.options.imagesPath+cardInfo.imageUrl+'" class="card front">');
				$a.append($img);
				var $backimg=$('<span class="card back"></span>');
				$a.append($backimg);
				$backimg.css({
					width: this.options.cardWidth,
					height: this.options.cardHeight
				});
				this._setCardStatus($container, this.disclosed[cardIndex]?2:0, true);
				$a.click(function(){
					var $card=$(this).parents(".memory-card-container");
					return $card.data("game")._cardClicked($card.get()[0]);
				});
				
				$(this.innerElement).append($container);
				if (this.disclosed[cardIndex]) this.enableCardLink($container);
			}
		},
		
		getCardId: function(cardIndex, side) {
			return this.elementId+"-card"+cardIndex+"-"+side;
		},
		
		_setCardStatus: function($card, status, store) {
			if (store) $card.data("status", status);
			var $frontImg=$card.find("img.front");
			var $backImg=$card.find(".card.back");
			var $a=$card.find("a");
			if (status>0) {
				$a.removeClass("back");
				$a.addClass("front");
			}
			else {
				$a.removeClass("front");
				$a.addClass("back");
			}
		},
		
		_cardClicked: function(card) {
			var $card=$(card);
			if ($card.data("currentDirection")!=0 || this.currentCards.length>1) return false;
			var cardStatus=$card.data("status");
			switch (cardStatus) {
				case 2:
					var href=$card.find("a").attr("href");
					if (href!==undefined && href!=null && href!="") return true;
					else return false;
				break;
				case 1:
					return false;
				break;
				case 0:
					this.nmoves++;
					this._startFlip(card,1);
					return false;
				break;
				
			}
		},
		
		_startFlip: function(card, direction) {
			var $card=$(card);
			//$card.stop(true);
			$card.data("currentDirection", direction);
			if (direction>0) {
				this.currentCards.push(card);
				$card.data("status",1);
			}
			else {
				
			}
			this._startFlipAnimation(card, direction);
		
		},
		
		_startFlipAnimation: function(card, direction) {
			var $card=$(card);
			
			var easing=this.options.flipAnimationEasing;
			
			var $a=$card.find("a");
			$a.animateScaleX(1, 0, this.options.cardFlipDuration/2, easing=="linear"?"linear":"easeIn"+easing, function(){
				var $card=$(this).parents(".memory-card-container");
				var game=$card.data("game");
				var direction=$card.data("currentDirection");
				if (direction>0) {
					$(this).addClass("front");
					$(this).removeClass("back");
				}
				else {
					$(this).addClass("back");
					$(this).removeClass("front");
				}
				
					
				$(this).animateScaleX(0, 1, game.option("cardFlipDuration")/2, easing=="linear"?"linear":"easeOut"+easing, function(){
					var $card=$(this).parents(".memory-card-container");
					var game=$card.data("game");
					var direction=$card.data("currentDirection");
					var cardIndex=$card.data("cardIndex");
					var event=false;
					if (direction>0) {
						var disclosed=false;
						if (game.getCurrentCardsLength()>0) { 
							var $currentCard0=$(game.getCurrentCard(0));
							if (game.getCurrentCardsLength()>1) {
								var $currentCard1=$(game.getCurrentCard(1));
								if ($currentCard1.data("cardIndex")==$currentCard0.data("cardIndex")) {
									disclosed=true;
									if (cardIndex!=$currentCard1.data("cardIndex")) {
										throw "Flipping with direction 1 ended on a card that is not within currentCards. Something's wrong.";
									}
									$currentCard1.data("status", 2);
									$currentCard0.data("status", 2);
									game.enableCardLink($currentCard0);
									game.enableCardLink($currentCard1);
									game.setDisclosed(cardIndex);
									game.currentCards=[];
									event=true;
								}
								else {
									game.closeCurrentCards();
								}
								
							}
						}
						    
					}
					else {
						$card.data("status", 0);
						game.popCurrentCard($card);
						if (game.resetPending!==null) {
							game.reset(false, game.resetPending.reorder, game.resetPending.rebuild);
						}
					}
					
					$card.data("currentDirection", 0);
					
					if (event) {
						var card=this;
						setTimeout(function(){
							game.option("onPairDisclosed").call(game, {
								card: card,
								cardIndex: cardIndex,
								cardInfo: game.option("cards")[cardIndex],
								disclosedPairs: game.ndisclosed,
								totalPairs: game.option("cards").length,
								moves: game.nmoves,
								finished: (game.ndisclosed==game.option("cards").length)
							});
						},1)
					}
				
				});
					
				
			});
				
			
		},
		
		getCurrentCardsLength: function() {
			return this.currentCards.length;
		},
		getCurrentCard: function(i) {
			return this.currentCards[i];
		},
		setDisclosed: function(idx) {
			if (!this.disclosed[idx]) this.ndisclosed++;
			this.disclosed[idx]=true;
		},
		
		enableCardLink: function($card) {
			var $a=$card.find("a");
			$a.attr("href", $card.data("linkUrl"));
			var linkTitle=$card.data("linkTitle");
			if (linkTitle!==undefined && linkTitle!==null) $a.attr("title", linkTitle);
		},
		popCurrentCard: function($card) {
			var nclosed=0;
			for (var i=0; i<this.currentCards.length; i++) {
				var $cc=$(this.currentCards[i]);
				if ($cc.data("status")==0) {
					nclosed++;
				}
			}
			if (nclosed==this.currentCards.length) this.currentCards=[];
		},
		
		closeCurrentCards: function() {
			if (this.timeoutId) clearTimeout(this.timeoutId);
			var that=this;
			this.timeoutId=setTimeout(function(){that.actuallyCloseCurrentCards();}, that.option("flipBackTimeout"));
		},
		
		actuallyCloseCurrentCards: function() {
			for (var i=0; i<this.currentCards.length; i++) {
				this._startFlip(this.currentCards[i],-1);
			}
		}
		
	
	
	});
	
	function debug(){
		if (window.console) console.log.apply(console,arguments);
	}

}(jQuery));