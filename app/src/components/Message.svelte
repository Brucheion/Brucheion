<script>
  import growAndFade from '../transitions/growAndFade'

  export let text = ''
  export let error = false
</script>

<style>
  em {
    display: inline-block;
    padding: 5px;

    font-style: normal;
    font-size: 90%;
    font-weight: 500;
  }

  .error {
    color: rgb(200, 10, 10);
  }
</style>

<div transition:growAndFade>
  <em class:error>{text}</em>
</div>
