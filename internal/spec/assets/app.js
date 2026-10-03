(function(){
  // Own-answer text selects the "own answer" option.
  var own=document.getElementById('own');
  if(own)own.addEventListener('input',function(){var r=document.getElementById('opt-own');if(r)r.checked=true});
})();
