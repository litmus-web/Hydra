# Sandman Examples
Ive added some examples as and when i test areas of Sandman and it's implementation.<br>

**Layout:**
  - `workers`: This is just a copy of the workers dir so its easy to test
  - `*.py`: Any test files, things like Django will have their own dir (Subject to change)
  
 ### Examples
 - The only working example is the Flask example which has *limited* use as of right now.<br> 
 - The WSGI adapter uses PyUv to let us run sync functions in a semi concurrent manor.
