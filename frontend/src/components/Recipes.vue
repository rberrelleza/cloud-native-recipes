<template>
  <div>
    <h1>{{ title }}</h1>

    <section v-if="errored">
      <p>We're sorry, we're not able to retrieve this information at the moment, please try back later</p>
    </section>

    <section v-else>
    <div v-if="loading">Loading...</div>
    <div v-else class="gallery">
      <div
        v-for="recipe in recipes"
        :key="recipe.id"
        class="gallery-panel"
      >
        <p><b>{{ recipe.title }}</b></p>
        <img :src="`${recipe.image}`" class="recipe"/>
        <div>
          <span class="icon is-small">
            <button v-on:click="upVote($event, recipe.id)" class="fa fa-chevron-up" />
              <strong class="has-text-info">{{ recipe.upVotes }}</strong>
          </span>

          <span class="icon is-small">
            <button @click="downVote($event, recipe.id)" class="fa fa-chevron-down" />
              <strong class="has-text-info">{{ recipe.downVotes }}</strong>
          </span>

        </div>
      </div>
    </div>
  </section>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'Recipes',
  data () {
    return {
      recipes: [],
      loading: true,
      errored: false,
      title: 'Cloud Native Recipes'
    }
  },
  mounted () {
    axios
      .get('/api/recipes')
      .then(response => {
        this.recipes = response.data
      })
      .catch(error => {
        console.log(error)
        this.errored = true
      })
      .finally(() => {
        this.loading = false
      })
  },
  methods: {
    upVote: function (e, id) {
      e.preventDefault()
      axios
        .post(`/api/recipes/${id}/up`)
        .then((response) => {
          const ix = this.recipes.findIndex(r => r.id === id)
          if (ix >= 0) {
            this.recipes[ix].upVotes = response.data.upVotes
          }
        })
        .catch(error => {
          console.log(error)
          this.errored = true
        })
        .finally(() => {

        })
    },
    downVote: function (e, id) {
      e.preventDefault()
      axios
        .post(`/api/recipes/${id}/down`)
        .then((response) => {
          console.log(response.data)
          const ix = this.recipes.findIndex(r => r.id === id)
          if (ix >= 0) {
            this.recipes[ix].downVotes = response.data.downVotes
          }
        })
        .catch(error => {
          console.log(error)
          this.errored = true
        })
        .finally(() => {

        })
    }
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
h1, h2 {
  font-weight: normal;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}

img.recipe {
  max-width: 400px;
}

.gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr));
  grid-gap: 1rem;
  max-width: 80rem;
  margin: 5rem auto;
  padding: 0 5rem;
}
.gallery-panel img {
  width: 100%;
  height: 22vw;
  object-fit: cover;
  border-radius: 0.75rem;
}

</style>
